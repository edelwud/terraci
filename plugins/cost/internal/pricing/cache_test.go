package pricing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/plugins/diskblob"
)

var (
	awsProviderID = "aws"
	awsServiceEC2 = ServiceID{Provider: awsProviderID, Name: "AmazonEC2"}
	awsServiceRDS = ServiceID{Provider: awsProviderID, Name: "AmazonRDS"}
)

// stubFetcher is a no-op fetcher for cache tests that don't need real pricing data.
type stubFetcher struct{}

func (s *stubFetcher) FetchRegionIndex(_ context.Context, _ ServiceID, _ string) (*PriceIndex, error) {
	return nil, fmt.Errorf("stub fetcher: not implemented")
}

// newTestCache builds a Cache from a raw diskblob store — mirrors the deleted NewCache constructor.
func newTestCache(store *diskblob.Store, ttl time.Duration, fetcher PriceFetcher) *Cache {
	if ttl == 0 {
		ttl = DefaultCacheTTL
	}
	c := NewCacheFromBlobCache(blobcache.New(store, "", ttl))
	if fetcher != nil {
		c.SetFetcher(fetcher)
	}
	return c
}

func TestNewCache_Defaults(t *testing.T) {
	store := diskblob.NewStore(t.TempDir())
	c := newTestCache(store, 0, &stubFetcher{})

	if c.blobs == nil {
		t.Fatal("expected blob cache to be initialized")
	}
	if c.fetcher.Load() == nil {
		t.Error("expected fetcher to be non-nil")
	}
}

func TestNewCache_Custom(t *testing.T) {
	tmpDir := t.TempDir()
	ttl := 1 * time.Hour

	c := newTestCache(diskblob.NewStore(tmpDir), ttl, &stubFetcher{})

	if c.ttl() != ttl {
		t.Errorf("expected ttl %v, got %v", ttl, c.ttl())
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name string
		idx  *PriceIndex
		ttl  time.Duration
		want bool
	}{
		{
			name: "nil index",
			idx:  nil,
			ttl:  time.Hour,
			want: false,
		},
		{
			name: "fresh index",
			idx: &PriceIndex{
				UpdatedAt: time.Now(),
			},
			ttl:  time.Hour,
			want: true,
		},
		{
			name: "expired index",
			idx: &PriceIndex{
				UpdatedAt: time.Now().Add(-48 * time.Hour),
			},
			ttl:  time.Hour,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cache{}
			if tt.ttl > 0 {
				c = newTestCache(diskblob.NewStore(t.TempDir()), tt.ttl, &stubFetcher{})
			}
			got := c.isFresh(tt.idx)
			if got != tt.want {
				t.Errorf("isValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	c := newTestCache(diskblob.NewStore(tmpDir), DefaultCacheTTL, &stubFetcher{})

	now := time.Now().Truncate(time.Second)
	idx := &PriceIndex{
		ServiceID: awsServiceEC2,
		Region:    "us-east-1",
		Version:   "v1.0",
		UpdatedAt: now,
		Products: map[string]Price{
			"SKU123": {
				SKU:           "SKU123",
				ProductFamily: "Compute Instance",
				Attributes:    map[string]string{"instanceType": "m5.large"},
				OnDemandUSD:   0.096,
				Unit:          "Hrs",
			},
		},
		Attributes: map[string]string{"format": "test"},
	}

	if err := c.saveToCache(context.Background(), idx); err != nil {
		t.Fatalf("saveToCache() error: %v", err)
	}

	loaded, err := c.loadFromCache(context.Background(), awsServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("loadFromCache() error: %v", err)
	}

	if loaded.ServiceID != idx.ServiceID {
		t.Errorf("ServiceID = %v, want %v", loaded.ServiceID, idx.ServiceID)
	}
	if loaded.Region != idx.Region {
		t.Errorf("Region = %v, want %v", loaded.Region, idx.Region)
	}
	if loaded.Version != idx.Version {
		t.Errorf("Version = %v, want %v", loaded.Version, idx.Version)
	}
	if !loaded.UpdatedAt.Equal(idx.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", loaded.UpdatedAt, idx.UpdatedAt)
	}
	if len(loaded.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(loaded.Products))
	}
	p := loaded.Products["SKU123"]
	if p.OnDemandUSD != 0.096 {
		t.Errorf("OnDemandUSD = %v, want 0.096", p.OnDemandUSD)
	}
	if p.Unit != "Hrs" {
		t.Errorf("Unit = %v, want Hrs", p.Unit)
	}
	if loaded.Attributes["format"] != "test" {
		t.Errorf("Attributes[format] = %v, want test", loaded.Attributes["format"])
	}
}

func TestLoadFromCache_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	c := newTestCache(diskblob.NewStore(tmpDir), DefaultCacheTTL, &stubFetcher{})

	_, err := c.loadFromCache(context.Background(), ServiceID{Provider: awsProviderID, Name: "NoSuchService"}, "no-region")
	if err == nil {
		t.Error("expected error for non-existent cache file, got nil")
	}
}

func TestInvalidate(t *testing.T) {
	tmpDir := t.TempDir()
	c := newTestCache(diskblob.NewStore(tmpDir), DefaultCacheTTL, &stubFetcher{})

	idx := &PriceIndex{
		ServiceID: awsServiceEC2,
		Region:    "us-west-2",
		UpdatedAt: time.Now(),
		Products:  map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
	}

	if err := c.saveToCache(context.Background(), idx); err != nil {
		t.Fatalf("saveToCache() error: %v", err)
	}

	if err := c.Invalidate(context.Background(), awsServiceEC2, "us-west-2"); err != nil {
		t.Fatalf("Invalidate() error: %v", err)
	}

	_, err := c.loadFromCache(context.Background(), awsServiceEC2, "us-west-2")
	if err == nil {
		t.Error("expected error after Invalidate, got nil")
	}
}

func TestValidate_AllCached(t *testing.T) {
	tmpDir := t.TempDir()
	c := newTestCache(diskblob.NewStore(tmpDir), DefaultCacheTTL, &stubFetcher{})

	idx := &PriceIndex{
		ServiceID: awsServiceEC2,
		Region:    "us-east-1",
		UpdatedAt: time.Now(),
		Products:  map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
	}
	if err := c.saveToCache(context.Background(), idx); err != nil {
		t.Fatalf("saveToCache() error: %v", err)
	}

	missing := c.Validate(context.Background(), map[ServiceID][]string{
		awsServiceEC2: {"us-east-1"},
	})

	if len(missing) != 0 {
		t.Errorf("expected no missing entries, got %d", len(missing))
	}
}

func TestValidate_SomeMissing(t *testing.T) {
	tmpDir := t.TempDir()
	c := newTestCache(diskblob.NewStore(tmpDir), DefaultCacheTTL, &stubFetcher{})

	idx := &PriceIndex{
		ServiceID: awsServiceEC2,
		Region:    "us-east-1",
		UpdatedAt: time.Now(),
		Products:  map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
	}
	if err := c.saveToCache(context.Background(), idx); err != nil {
		t.Fatalf("saveToCache() error: %v", err)
	}

	missing := c.Validate(context.Background(), map[ServiceID][]string{
		awsServiceEC2: {"us-east-1", "eu-west-1"},
	})

	if len(missing) != 1 {
		t.Fatalf("expected 1 missing entry, got %d", len(missing))
	}
	if missing[0].Service != awsServiceEC2 || missing[0].Region != "eu-west-1" {
		t.Errorf("unexpected missing entry: %+v", missing[0])
	}
}

// fakeFetcher returns PriceIndex directly without HTTP, for tests needing fetch behavior.
type fakeFetcher struct {
	index *PriceIndex
	err   error
}

func (f *fakeFetcher) FetchRegionIndex(_ context.Context, _ ServiceID, _ string) (*PriceIndex, error) {
	return f.index, f.err
}

func newTestFetcher() *fakeFetcher {
	return &fakeFetcher{
		index: &PriceIndex{
			ServiceID: awsServiceEC2,
			Region:    "us-east-1",
			Version:   "test",
			UpdatedAt: time.Now(),
			Products: map[string]Price{
				"SKU1": {
					SKU:           "SKU1",
					ProductFamily: "Compute Instance",
					Attributes:    map[string]string{"instanceType": "t3.micro", "location": "US East (N. Virginia)"},
					OnDemandUSD:   0.0104,
					Unit:          "Hrs",
				},
			},
		},
	}
}

func TestGetIndex_CacheHit(t *testing.T) {
	tmpDir := t.TempDir()
	c := newTestCache(diskblob.NewStore(tmpDir), time.Hour, &stubFetcher{})

	// Pre-populate cache
	idx := &PriceIndex{
		ServiceID: awsServiceEC2,
		Region:    "us-east-1",
		Version:   "cached",
		UpdatedAt: time.Now(),
		Products:  map[string]Price{"SKU1": {SKU: "SKU1", OnDemandUSD: 0.01}},
	}
	if err := c.saveToCache(context.Background(), idx); err != nil {
		t.Fatalf("saveToCache: %v", err)
	}

	got, err := c.GetIndex(context.Background(), awsServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("GetIndex: %v", err)
	}
	if got.Version != "cached" {
		t.Errorf("expected cached version, got %q", got.Version)
	}
}

func TestGetIndex_CacheMiss(t *testing.T) {
	tmpDir := t.TempDir()
	c := newTestCache(diskblob.NewStore(tmpDir), time.Hour, newTestFetcher())

	got, err := c.GetIndex(context.Background(), awsServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("GetIndex: %v", err)
	}
	if got.Version != "test" {
		t.Errorf("expected fetched version 'test', got %q", got.Version)
	}

	// Verify it was saved to cache
	loaded, err := c.loadFromCache(context.Background(), awsServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("expected cache to be populated: %v", err)
	}
	if loaded.Version != "test" {
		t.Errorf("cached version = %q, want 'test'", loaded.Version)
	}
}

func TestGetIndex_FetchError(t *testing.T) {
	tmpDir := t.TempDir()
	errFetcher := &fakeFetcher{err: fmt.Errorf("network error")}
	c := newTestCache(diskblob.NewStore(tmpDir), time.Hour, errFetcher)

	_, err := c.GetIndex(context.Background(), awsServiceEC2, "us-east-1")
	if err == nil {
		t.Error("expected error on fetch failure")
	}
}

func TestCache_SetFetcher(t *testing.T) {
	c := newTestCache(diskblob.NewStore(t.TempDir()), 2*time.Hour, &stubFetcher{})

	t.Run("SetFetcher", func(t *testing.T) {
		f := &stubFetcher{}
		c.SetFetcher(f)
		loaded := c.fetcher.Load()
		if loaded == nil || *loaded != PriceFetcher(f) {
			t.Error("SetFetcher did not set fetcher")
		}
	})
}

func TestGetIndex_StaleFallback(t *testing.T) {
	tmpDir := t.TempDir()
	ttl := 1 * time.Hour
	c := newTestCache(diskblob.NewStore(tmpDir), ttl, &stubFetcher{})

	// Save a valid index to cache
	idx := &PriceIndex{
		ServiceID: awsServiceEC2,
		Region:    "us-east-1",
		Version:   "stale-version",
		UpdatedAt: time.Now(),
		Products:  map[string]Price{"SKU1": {SKU: "SKU1", OnDemandUSD: 0.01}},
	}
	if err := c.saveToCache(context.Background(), idx); err != nil {
		t.Fatalf("saveToCache: %v", err)
	}

	idx.UpdatedAt = time.Now().Add(-2 * time.Hour)
	if err := c.saveToCache(context.Background(), idx); err != nil {
		t.Fatalf("saveToCache stale: %v", err)
	}

	// Replace fetcher with one that fails — should fall back to stale cache
	c.SetFetcher(&fakeFetcher{err: fmt.Errorf("network unreachable")})

	got, err := c.GetIndex(context.Background(), awsServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("GetIndex should succeed with stale fallback, got error: %v", err)
	}
	if got.Version != "stale-version" {
		t.Errorf("expected stale version, got %q", got.Version)
	}
}

func TestGetIndex_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	var fetchCount atomic.Int32
	fetcher := &fakeFetcher{
		index: newTestFetcher().index,
	}
	c := newTestCache(diskblob.NewStore(tmpDir), time.Hour, PriceFetcherFunc(func(ctx context.Context, service ServiceID, region string) (*PriceIndex, error) {
		fetchCount.Add(1)
		time.Sleep(10 * time.Millisecond)
		return fetcher.FetchRegionIndex(ctx, service, region)
	}))

	// Run 10 goroutines fetching the same service/region concurrently
	const goroutines = 10
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			_, err := c.GetIndex(context.Background(), awsServiceEC2, "us-east-1")
			errs <- err
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("concurrent GetIndex error: %v", err)
		}
	}

	if got := fetchCount.Load(); got != 1 {
		t.Errorf("fetchCount = %d, want 1 for deduplicated concurrent access", got)
	}
}

func TestGetIndex_ConcurrentWaiterRespectsContext(t *testing.T) {
	tmpDir := t.TempDir()
	var fetchCount atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})

	c := newTestCache(diskblob.NewStore(tmpDir), time.Hour, PriceFetcherFunc(func(_ context.Context, service ServiceID, region string) (*PriceIndex, error) {
		fetchCount.Add(1)
		close(started)
		<-release
		return &PriceIndex{
			ServiceID: service,
			Region:    region,
			Version:   "test",
			UpdatedAt: time.Now(),
			Products:  map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
		}, nil
	}))

	leaderErr := make(chan error, 1)
	go func() {
		_, err := c.GetIndex(context.Background(), awsServiceEC2, "us-east-1")
		leaderErr <- err
	}()

	<-started

	waiterCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	if _, err := c.GetIndex(waiterCtx, awsServiceEC2, "us-east-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiter GetIndex error = %v, want %v", err, context.DeadlineExceeded)
	}

	close(release)
	if err := <-leaderErr; err != nil {
		t.Fatalf("leader GetIndex error: %v", err)
	}
	if got := fetchCount.Load(); got != 1 {
		t.Fatalf("fetchCount = %d, want 1", got)
	}
}

func TestGetIndex_ConcurrentDifferentKeysFetchIndependently(t *testing.T) {
	tmpDir := t.TempDir()
	var fetchCount atomic.Int32
	c := newTestCache(diskblob.NewStore(tmpDir), time.Hour, PriceFetcherFunc(func(_ context.Context, service ServiceID, region string) (*PriceIndex, error) {
		fetchCount.Add(1)
		return &PriceIndex{
			ServiceID: service,
			Region:    region,
			Version:   "test",
			UpdatedAt: time.Now(),
			Products:  map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
		}, nil
	}))

	var wg sync.WaitGroup
	requests := []struct {
		service ServiceID
		region  string
	}{
		{service: awsServiceEC2, region: "us-east-1"},
		{service: awsServiceEC2, region: "us-west-2"},
		{service: awsServiceRDS, region: "us-east-1"},
	}

	for _, req := range requests {
		wg.Go(func() {
			if _, err := c.GetIndex(context.Background(), req.service, req.region); err != nil {
				t.Errorf("GetIndex(%s,%s) error: %v", req.service, req.region, err)
			}
		})
	}
	wg.Wait()

	if got := fetchCount.Load(); got != int32(len(requests)) {
		t.Fatalf("fetchCount = %d, want %d for distinct keys", got, len(requests))
	}
}

type PriceFetcherFunc func(ctx context.Context, service ServiceID, region string) (*PriceIndex, error)

func (f PriceFetcherFunc) FetchRegionIndex(ctx context.Context, service ServiceID, region string) (*PriceIndex, error) {
	return f(ctx, service, region)
}

func TestGetIndex_FetchError_NoStaleCache(t *testing.T) {
	tmpDir := t.TempDir()
	errFetcher := &fakeFetcher{err: fmt.Errorf("network error")}
	c := newTestCache(diskblob.NewStore(tmpDir), time.Hour, errFetcher)

	// No pre-populated cache, fetch fails — should return error
	_, err := c.GetIndex(context.Background(), awsServiceEC2, "us-east-1")
	if err == nil {
		t.Error("expected error when fetch fails and no stale cache exists")
	}
}
