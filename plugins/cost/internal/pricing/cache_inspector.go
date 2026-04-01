package pricing

import (
	"context"
	"strings"
	"time"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
)

// CacheEntry describes a single cached pricing object.
type CacheEntry struct {
	Service   ServiceID
	Region    string
	Age       time.Duration
	ExpiresIn time.Duration // negative if expired
}

// CacheInspector adapts generic blob cache objects into pricing-oriented diagnostics.
type CacheInspector struct {
	blobs *blobcache.Cache
}

// NewCacheInspector creates a pricing-oriented cache inspector over a prepared blob cache.
func NewCacheInspector(blobs *blobcache.Cache) *CacheInspector {
	return &CacheInspector{blobs: blobs}
}

// Dir returns the resolved blob cache directory when exposed by the backend.
func (i *CacheInspector) Dir() string {
	if i == nil || i.blobs == nil {
		return ""
	}
	return i.blobs.Dir()
}

// TTL returns the configured blob cache TTL.
func (i *CacheInspector) TTL() time.Duration {
	if i == nil || i.blobs == nil {
		return 0
	}
	return i.blobs.TTL()
}

// Entries returns pricing cache entries parsed from blob cache objects.
func (i *CacheInspector) Entries(ctx context.Context) []CacheEntry {
	if i == nil || i.blobs == nil {
		return nil
	}

	objects, err := i.blobs.List(ctx)
	if err != nil {
		log.WithError(err).Debug("pricing cache list failed")
		return nil
	}

	entries := make([]CacheEntry, 0, len(objects))
	for _, object := range objects {
		service, region, ok := parseCacheKey(object.Key)
		if !ok {
			continue
		}
		entries = append(entries, CacheEntry{
			Service:   service,
			Region:    region,
			Age:       object.Age,
			ExpiresIn: object.ExpiresIn,
		})
	}

	return entries
}

// OldestAge returns the age of the oldest cached entry, or 0 if cache is empty.
func (i *CacheInspector) OldestAge(ctx context.Context) time.Duration {
	entries := i.Entries(ctx)
	oldest := time.Duration(0)
	for _, entry := range entries {
		if oldest == 0 || entry.Age > oldest {
			oldest = entry.Age
		}
	}
	return oldest
}

func parseCacheKey(key string) (ServiceID, string, bool) {
	parts := strings.Split(key, "/")
	if len(parts) != 3 || !strings.HasSuffix(parts[2], ".json") {
		return ServiceID{}, "", false
	}

	return ServiceID{
		Provider: parts[0],
		Name:     parts[1],
	}, strings.TrimSuffix(parts[2], ".json"), true
}
