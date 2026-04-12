package engine

import (
	"context"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

// CacheInspector provides diagnostic and maintenance access to the pricing cache.
// Obtain it via Estimator.Cache().
type CacheInspector interface {
	// Dir returns the resolved pricing cache directory path, or empty if not backed by disk.
	Dir() string
	// TTL returns the cache time-to-live for pricing entries.
	TTL() time.Duration
	// OldestAge returns the age of the oldest cache entry, or 0 if the cache is empty.
	OldestAge(ctx context.Context) time.Duration
	// Entries returns info about all cached pricing files.
	Entries(ctx context.Context) []pricing.CacheEntry
	// CleanExpired removes all expired cache entries.
	CleanExpired(ctx context.Context)
}

// cacheInspector implements CacheInspector backed by EstimationRuntime.
type cacheInspector struct {
	r *costruntime.EstimationRuntime
}

func (c *cacheInspector) Dir() string                                 { return c.r.CacheDir() }
func (c *cacheInspector) TTL() time.Duration                          { return c.r.CacheTTL() }
func (c *cacheInspector) OldestAge(ctx context.Context) time.Duration { return c.r.CacheOldestAge(ctx) }
func (c *cacheInspector) Entries(ctx context.Context) []pricing.CacheEntry {
	return c.r.CacheEntries(ctx)
}
func (c *cacheInspector) CleanExpired(ctx context.Context) { c.r.CleanExpiredCache(ctx) }
