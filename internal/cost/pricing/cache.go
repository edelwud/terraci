package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/caarlos0/log"
)

const (
	// DefaultCacheDir is the default cache directory name
	DefaultCacheDir = ".terraci/pricing"
	// DefaultCacheTTL is how long cached data is considered valid
	DefaultCacheTTL = 24 * time.Hour
)

// Cache manages local pricing data cache
type Cache struct {
	dir     string
	ttl     time.Duration
	fetcher *Fetcher
}

// NewCache creates a new pricing cache
func NewCache(cacheDir string, ttl time.Duration) *Cache {
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
		dir:     cacheDir,
		ttl:     ttl,
		fetcher: NewFetcher(),
	}
}

// GetIndex returns a pricing index for a service/region, using cache if valid
func (c *Cache) GetIndex(ctx context.Context, service ServiceCode, region string) (*PriceIndex, error) {
	// Try cache first
	idx, err := c.loadFromCache(service, region)
	if err == nil && c.isValid(idx) {
		log.WithField("service", string(service)).
			WithField("region", region).
			Debug("using cached pricing data")
		return idx, nil
	}

	// Fetch fresh data
	log.WithField("service", string(service)).
		WithField("region", region).
		Info("downloading pricing data from AWS")

	idx, err = c.fetcher.FetchRegionIndex(ctx, service, region)
	if err != nil {
		return nil, err
	}

	// Save to cache
	if saveErr := c.saveToCache(idx); saveErr != nil {
		log.WithError(saveErr).Warn("failed to save pricing cache")
	}

	return idx, nil
}

// Invalidate removes cached data for a service/region
func (c *Cache) Invalidate(service ServiceCode, region string) error {
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
func (c *Cache) Validate(services map[ServiceCode][]string) []struct {
	Service ServiceCode
	Region  string
} {
	var missing []struct {
		Service ServiceCode
		Region  string
	}

	for service, regions := range services {
		for _, region := range regions {
			idx, err := c.loadFromCache(service, region)
			if err != nil || !c.isValid(idx) {
				missing = append(missing, struct {
					Service ServiceCode
					Region  string
				}{service, region})
			}
		}
	}

	return missing
}

// PrewarmCache downloads and caches pricing data for specified services/regions
func (c *Cache) PrewarmCache(ctx context.Context, services map[ServiceCode][]string) error {
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
func (c *Cache) cachePath(service ServiceCode, region string) string {
	return filepath.Join(c.dir, string(service), region+".json")
}

// isValid checks if cached data is still valid
func (c *Cache) isValid(idx *PriceIndex) bool {
	if idx == nil {
		return false
	}
	return time.Since(idx.UpdatedAt) < c.ttl
}

// loadFromCache loads a cached index
func (c *Cache) loadFromCache(service ServiceCode, region string) (*PriceIndex, error) {
	path := c.cachePath(service, region)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var idx PriceIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}

	return &idx, nil
}

// saveToCache saves an index to cache
func (c *Cache) saveToCache(idx *PriceIndex) error {
	path := c.cachePath(idx.ServiceCode, idx.Region)

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
		if filepath.Ext(path) != ".json" {
			return nil
		}

		// Check if file is older than TTL
		if time.Since(info.ModTime()) > c.ttl {
			log.WithField("path", path).Debug("removing expired cache")
			return os.Remove(path)
		}
		return nil
	})
}
