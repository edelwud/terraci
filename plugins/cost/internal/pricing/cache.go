package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/plugin"
)

const (
	// DefaultCacheTTL is how long cached data is considered valid.
	DefaultCacheTTL = 24 * time.Hour
)

// PriceFetcher abstracts pricing data retrieval.
// Implemented by Fetcher (AWS) and potentially GCP/Azure fetchers.
type PriceFetcher interface {
	FetchRegionIndex(ctx context.Context, service ServiceID, region string) (*PriceIndex, error)
}

// Cache manages pricing data over a pluggable blob store.
// Safe for concurrent use.
type Cache struct {
	blobs      *blobcache.Cache
	fetcher    PriceFetcher
	inflightMu sync.Mutex
	inflight   map[string]*inflightFetch
}

type inflightFetch struct {
	done chan struct{}
	idx  *PriceIndex
	err  error
}

// NewCache creates a new pricing cache using the given blob store.
func NewCache(store plugin.BlobStore, namespace string, ttl time.Duration, fetcher PriceFetcher) *Cache {
	if ttl == 0 {
		ttl = DefaultCacheTTL
	}

	return NewCacheFromBlobCache(blobcache.New(store, namespace, ttl), fetcher)
}

// NewCacheFromBlobCache creates a new pricing cache over a prepared blob cache.
func NewCacheFromBlobCache(blobs *blobcache.Cache, fetcher PriceFetcher) *Cache {
	return &Cache{
		blobs:    blobs,
		fetcher:  fetcher,
		inflight: make(map[string]*inflightFetch),
	}
}

// SetFetcher replaces the fetcher (used for testing or alternative providers).
func (c *Cache) SetFetcher(f PriceFetcher) { c.fetcher = f }

// GetIndex returns a pricing index for a service/region, using cache if valid.
func (c *Cache) GetIndex(ctx context.Context, service ServiceID, region string) (*PriceIndex, error) {
	idx, err := c.loadFromCache(ctx, service, region)
	if err == nil && c.isValid(idx) {
		log.WithField("service", service.String()).
			WithField("region", region).
			Debug("using cached pricing data")
		return idx, nil
	}

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

// Invalidate removes cached data for a service/region.
func (c *Cache) Invalidate(ctx context.Context, service ServiceID, region string) error {
	if c.blobs == nil {
		return nil
	}
	return c.blobs.Delete(ctx, c.cacheKey(service, region))
}

// MissingPricingEntry identifies a service/region combination absent from the cache.
type MissingPricingEntry struct {
	Service ServiceID
	Region  string
}

// Validate checks if required pricing data is cached and valid.
// Returns list of missing service/region combinations.
func (c *Cache) Validate(ctx context.Context, services map[ServiceID][]string) []MissingPricingEntry {
	var missing []MissingPricingEntry

	for service, regions := range services {
		for _, region := range regions {
			idx, err := c.loadFromCache(ctx, service, region)
			if err != nil || !c.isValid(idx) {
				missing = append(missing, MissingPricingEntry{service, region})
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
	if idx, err := c.loadFromCache(ctx, service, region); err == nil && c.isValid(idx) {
		return idx, nil
	}

	log.WithField("service", service.String()).
		WithField("region", region).
		Info("downloading pricing data")

	idx, err := c.fetcher.FetchRegionIndex(ctx, service, region)
	if err != nil {
		if stale, loadErr := c.loadFromCache(ctx, service, region); loadErr == nil && stale != nil {
			log.WithError(err).
				WithField("service", service.String()).
				WithField("region", region).
				Warn("fetch failed, using stale cache")
			return stale, nil
		}
		return nil, err
	}

	if saveErr := c.saveToCache(ctx, idx); saveErr != nil {
		log.WithError(saveErr).Warn("failed to save pricing cache")
	}

	return idx, nil
}

// PrewarmCache downloads and caches pricing data for specified services/regions.
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

func (c *Cache) cacheKey(service ServiceID, region string) string {
	return strings.Join([]string{service.Provider, service.Name, region + ".json"}, "/")
}

// isValid checks if cached data is still valid.
func (c *Cache) isValid(idx *PriceIndex) bool {
	if idx == nil {
		return false
	}
	return time.Since(idx.UpdatedAt) < c.ttl()
}

// loadFromCache loads a cached index from the blob store.
func (c *Cache) loadFromCache(ctx context.Context, service ServiceID, region string) (*PriceIndex, error) {
	if c.blobs == nil {
		return nil, blobcache.ErrStoreNotConfigured
	}

	data, _, ok, err := c.blobs.Get(ctx, c.cacheKey(service, region))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, blobcache.ErrEntryNotFound
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

// saveToCache saves an index to the blob store.
func (c *Cache) saveToCache(ctx context.Context, idx *PriceIndex) error {
	if c.blobs == nil {
		return blobcache.ErrStoreNotConfigured
	}

	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}

	opts := blobcache.PutOptions{
		ContentType: "application/json",
		Metadata: map[string]string{
			"provider": idx.ServiceID.Provider,
			"service":  idx.ServiceID.Name,
			"region":   idx.Region,
		},
	}
	if ttl := c.ttl(); ttl > 0 {
		expiresAt := time.Now().Add(ttl)
		opts.ExpiresAt = &expiresAt
	}

	_, err = c.blobs.Put(ctx, c.cacheKey(idx.ServiceID, idx.Region), data, opts)
	return err
}

func (c *Cache) ttl() time.Duration {
	if c.blobs == nil {
		return 0
	}
	return c.blobs.TTL()
}
