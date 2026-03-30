package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/caarlos0/log"
)

const (
	// DefaultCacheDir is the default cache directory name
	DefaultCacheDir = ".terraci/pricing"
	// DefaultCacheTTL is how long cached data is considered valid
	DefaultCacheTTL = 24 * time.Hour
	// cacheFileExt is the file extension for cached pricing files
	cacheFileExt = ".json"
)

// PriceFetcher abstracts pricing data retrieval.
// Implemented by Fetcher (AWS) and potentially GCP/Azure fetchers.
type PriceFetcher interface {
	FetchRegionIndex(ctx context.Context, service ServiceID, region string) (*PriceIndex, error)
}

// Cache manages local pricing data cache.
// Safe for concurrent use.
type Cache struct {
	dir        string
	ttl        time.Duration
	fetcher    PriceFetcher
	mu         sync.Mutex // protects file writes
	inflightMu sync.Mutex
	inflight   map[string]*inflightFetch
}

type inflightFetch struct {
	done chan struct{}
	idx  *PriceIndex
	err  error
}

// NewCache creates a new pricing cache with the given fetcher.
// The fetcher determines which cloud provider's pricing API is used.
func NewCache(cacheDir string, ttl time.Duration, fetcher PriceFetcher) *Cache {
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		cacheDir = filepath.Join(home, DefaultCacheDir)
	}
	if ttl == 0 {
		ttl = DefaultCacheTTL
	}
	return &Cache{
		dir:      cacheDir,
		ttl:      ttl,
		fetcher:  fetcher,
		inflight: make(map[string]*inflightFetch),
	}
}

// SetFetcher replaces the fetcher (used for testing or alternative providers).
func (c *Cache) SetFetcher(f PriceFetcher) { c.fetcher = f }

// Dir returns the resolved cache directory path (absolute).
func (c *Cache) Dir() string {
	if abs, err := filepath.Abs(c.dir); err == nil {
		return abs
	}
	return c.dir
}

// TTL returns the cache time-to-live duration.
func (c *Cache) TTL() time.Duration { return c.ttl }

// OldestAge returns the age of the oldest cached entry, or 0 if cache is empty.
func (c *Cache) OldestAge() time.Duration {
	var oldest time.Time
	_ = filepath.Walk(c.dir, func(path string, info os.FileInfo, walkErr error) error { //nolint:errcheck
		if walkErr != nil || info.IsDir() || filepath.Ext(path) != cacheFileExt {
			return nil //nolint:nilerr // skip errors, continue walking
		}
		if oldest.IsZero() || info.ModTime().Before(oldest) {
			oldest = info.ModTime()
		}
		return nil
	})
	if oldest.IsZero() {
		return 0
	}
	return time.Since(oldest)
}

// CacheEntry describes a single cached pricing file.
type CacheEntry struct {
	Service   ServiceID
	Region    string
	Age       time.Duration
	ExpiresIn time.Duration // negative if expired
}

// Entries returns info about all cached pricing files.
func (c *Cache) Entries() []CacheEntry {
	var entries []CacheEntry
	_ = filepath.Walk(c.dir, func(path string, info os.FileInfo, walkErr error) error { //nolint:errcheck
		if walkErr != nil || info.IsDir() || filepath.Ext(path) != cacheFileExt {
			return nil //nolint:nilerr
		}
		// Extract service/region from path: {dir}/{provider}/{service}/{region}.json
		rel, err := filepath.Rel(c.dir, path)
		if err != nil {
			return nil //nolint:nilerr
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 3 {
			return nil
		}
		service := ServiceID{
			Provider: parts[0],
			Name:     parts[1],
		}
		region := strings.TrimSuffix(parts[2], cacheFileExt)
		age := time.Since(info.ModTime())

		entries = append(entries, CacheEntry{
			Service:   service,
			Region:    region,
			Age:       age,
			ExpiresIn: c.ttl - age,
		})
		return nil
	})
	return entries
}

// GetIndex returns a pricing index for a service/region, using cache if valid
func (c *Cache) GetIndex(ctx context.Context, service ServiceID, region string) (*PriceIndex, error) {
	// Try cache first
	idx, err := c.loadFromCache(service, region)
	if err == nil && c.isValid(idx) {
		log.WithField("service", service.String()).
			WithField("region", region).
			Debug("using cached pricing data")
		return idx, nil
	}

	// Log why cache was not used
	if err != nil {
		log.WithField("service", service.String()).
			WithField("region", region).
			WithError(err).
			Debug("cache miss")
	} else if idx != nil {
		log.WithField("service", service.String()).
			WithField("region", region).
			WithField("age", time.Since(idx.UpdatedAt).Truncate(time.Minute)).
			Debug("cache expired")
	}

	return c.fetchAndCacheIndex(ctx, service, region)
}

// Invalidate removes cached data for a service/region
func (c *Cache) Invalidate(service ServiceID, region string) error {
	path := c.cachePath(service, region)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// InvalidateAll clears the entire cache
func (c *Cache) InvalidateAll() error {
	return os.RemoveAll(c.dir)
}

// Validate checks if required pricing data is cached and valid
// Returns list of missing service/region combinations
func (c *Cache) Validate(services map[ServiceID][]string) []struct {
	Service ServiceID
	Region  string
} {
	var missing []struct {
		Service ServiceID
		Region  string
	}

	for service, regions := range services {
		for _, region := range regions {
			idx, err := c.loadFromCache(service, region)
			if err != nil || !c.isValid(idx) {
				missing = append(missing, struct {
					Service ServiceID
					Region  string
				}{service, region})
			}
		}
	}

	return missing
}

func (c *Cache) fetchAndCacheIndex(ctx context.Context, service ServiceID, region string) (*PriceIndex, error) {
	key := c.cacheKey(service, region)

	c.inflightMu.Lock()
	if call, ok := c.inflight[key]; ok {
		c.inflightMu.Unlock()
		<-call.done
		return call.idx, call.err
	}

	call := &inflightFetch{done: make(chan struct{})}
	c.inflight[key] = call
	c.inflightMu.Unlock()

	call.idx, call.err = c.fetchAndCacheIndexLeader(ctx, service, region)

	c.inflightMu.Lock()
	delete(c.inflight, key)
	close(call.done)
	c.inflightMu.Unlock()

	return call.idx, call.err
}

func (c *Cache) fetchAndCacheIndexLeader(ctx context.Context, service ServiceID, region string) (*PriceIndex, error) {
	// One last cache check after winning the in-flight slot, so a previous leader
	// can satisfy late joiners without another remote fetch.
	if idx, err := c.loadFromCache(service, region); err == nil && c.isValid(idx) {
		return idx, nil
	}

	log.WithField("service", service.String()).
		WithField("region", region).
		Info("downloading pricing data")

	idx, err := c.fetcher.FetchRegionIndex(ctx, service, region)
	if err != nil {
		// If fetch fails and we have stale cache, use it as fallback
		if stale, loadErr := c.loadFromCache(service, region); loadErr == nil && stale != nil {
			log.WithError(err).
				WithField("service", service.String()).
				WithField("region", region).
				Warn("fetch failed, using stale cache")
			return stale, nil
		}
		return nil, err
	}

	if saveErr := c.saveToCache(idx); saveErr != nil {
		log.WithError(saveErr).Warn("failed to save pricing cache")
	}

	return idx, nil
}

// PrewarmCache downloads and caches pricing data for specified services/regions
func (c *Cache) PrewarmCache(ctx context.Context, services map[ServiceID][]string) error {
	for service, regions := range services {
		for _, region := range regions {
			if _, err := c.GetIndex(ctx, service, region); err != nil {
				return fmt.Errorf("prewarm %s/%s: %w", service, region, err)
			}
		}
	}
	return nil
}

// cachePath returns the cache file path for a service/region
func (c *Cache) cachePath(service ServiceID, region string) string {
	return filepath.Join(c.dir, service.Provider, service.Name, region+".json")
}

func (c *Cache) cacheKey(service ServiceID, region string) string {
	return service.String() + "|" + region
}

// isValid checks if cached data is still valid
func (c *Cache) isValid(idx *PriceIndex) bool {
	if idx == nil {
		return false
	}
	return time.Since(idx.UpdatedAt) < c.ttl
}

// loadFromCache loads a cached index
func (c *Cache) loadFromCache(service ServiceID, region string) (*PriceIndex, error) {
	path := c.cachePath(service, region)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var idx PriceIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}

	if !idx.isValid() {
		return nil, errors.New("invalid cache entry")
	}

	return &idx, nil
}

// saveToCache saves an index to cache (mutex-protected for concurrent access).
func (c *Cache) saveToCache(idx *PriceIndex) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := c.cachePath(idx.ServiceID, idx.Region)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// CleanExpired removes all expired cache entries
func (c *Cache) CleanExpired() error {
	return filepath.Walk(c.dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			// Log and skip files with access errors to continue cleaning other entries
			log.WithError(walkErr).WithField("path", path).Debug("skipping inaccessible file")
			// Intentionally return nil to continue walking, not propagate access errors
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != cacheFileExt {
			return nil
		}

		// Check if file is older than TTL
		if time.Since(info.ModTime()) > c.ttl {
			log.WithField("path", path).Debug("removing expired cache")
			return os.Remove(path) //nolint:gosec // path is constructed internally from cache dir
		}
		return nil
	})
}
