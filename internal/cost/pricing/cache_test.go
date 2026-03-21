package pricing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewCache_Defaults(t *testing.T) {
	c := NewCache("", 0)

	if c.ttl != DefaultCacheTTL {
		t.Errorf("expected ttl %v, got %v", DefaultCacheTTL, c.ttl)
	}
	if !strings.HasSuffix(c.dir, filepath.Join(".terraci", "pricing")) {
		t.Errorf("expected dir to end with .terraci/pricing, got %s", c.dir)
	}
	if c.fetcher == nil {
		t.Error("expected fetcher to be non-nil")
	}
}

func TestNewCache_Custom(t *testing.T) {
	tmpDir := t.TempDir()
	ttl := 1 * time.Hour

	c := NewCache(tmpDir, ttl)

	if c.dir != tmpDir {
		t.Errorf("expected dir %s, got %s", tmpDir, c.dir)
	}
	if c.ttl != ttl {
		t.Errorf("expected ttl %v, got %v", ttl, c.ttl)
	}
}

func TestCachePath(t *testing.T) {
	cacheDir := filepath.Join(os.TempDir(), "cache")
	c := &Cache{dir: cacheDir}

	got := c.cachePath("AmazonEC2", "us-east-1")
	want := filepath.Join(cacheDir, "AmazonEC2", "us-east-1.json")

	if got != want {
		t.Errorf("expected %s, got %s", want, got)
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
			c := &Cache{ttl: tt.ttl}
			got := c.isValid(tt.idx)
			if got != tt.want {
				t.Errorf("isValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	c := &Cache{dir: tmpDir, ttl: DefaultCacheTTL}

	now := time.Now().Truncate(time.Second)
	idx := &PriceIndex{
		ServiceCode: ServiceEC2,
		Region:      "us-east-1",
		Version:     "v1.0",
		UpdatedAt:   now,
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

	if err := c.saveToCache(idx); err != nil {
		t.Fatalf("saveToCache() error: %v", err)
	}

	loaded, err := c.loadFromCache(ServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("loadFromCache() error: %v", err)
	}

	if loaded.ServiceCode != idx.ServiceCode {
		t.Errorf("ServiceCode = %v, want %v", loaded.ServiceCode, idx.ServiceCode)
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
	c := &Cache{dir: tmpDir, ttl: DefaultCacheTTL}

	_, err := c.loadFromCache("NoSuchService", "no-region")
	if err == nil {
		t.Error("expected error for non-existent cache file, got nil")
	}
}

func TestInvalidate(t *testing.T) {
	tmpDir := t.TempDir()
	c := &Cache{dir: tmpDir, ttl: DefaultCacheTTL}

	idx := &PriceIndex{
		ServiceCode: ServiceEC2,
		Region:      "us-west-2",
		UpdatedAt:   time.Now(),
		Products:    map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
	}

	if err := c.saveToCache(idx); err != nil {
		t.Fatalf("saveToCache() error: %v", err)
	}

	if err := c.Invalidate(ServiceEC2, "us-west-2"); err != nil {
		t.Fatalf("Invalidate() error: %v", err)
	}

	_, err := c.loadFromCache(ServiceEC2, "us-west-2")
	if err == nil {
		t.Error("expected error after Invalidate, got nil")
	}
}

func TestInvalidateAll(t *testing.T) {
	tmpDir := t.TempDir()
	c := &Cache{dir: tmpDir, ttl: DefaultCacheTTL}

	services := []struct {
		code   ServiceCode
		region string
	}{
		{ServiceEC2, "us-east-1"},
		{ServiceRDS, "eu-west-1"},
	}

	for _, s := range services {
		idx := &PriceIndex{
			ServiceCode: s.code,
			Region:      s.region,
			UpdatedAt:   time.Now(),
			Products:    map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
		}
		if err := c.saveToCache(idx); err != nil {
			t.Fatalf("saveToCache(%s, %s) error: %v", s.code, s.region, err)
		}
	}

	if err := c.InvalidateAll(); err != nil {
		t.Fatalf("InvalidateAll() error: %v", err)
	}

	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Errorf("expected cache dir to be removed, stat error: %v", err)
	}
}

func TestValidate_AllCached(t *testing.T) {
	tmpDir := t.TempDir()
	c := &Cache{dir: tmpDir, ttl: DefaultCacheTTL}

	idx := &PriceIndex{
		ServiceCode: ServiceEC2,
		Region:      "us-east-1",
		UpdatedAt:   time.Now(),
		Products:    map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
	}
	if err := c.saveToCache(idx); err != nil {
		t.Fatalf("saveToCache() error: %v", err)
	}

	missing := c.Validate(map[ServiceCode][]string{
		ServiceEC2: {"us-east-1"},
	})

	if len(missing) != 0 {
		t.Errorf("expected no missing entries, got %d", len(missing))
	}
}

func TestValidate_SomeMissing(t *testing.T) {
	tmpDir := t.TempDir()
	c := &Cache{dir: tmpDir, ttl: DefaultCacheTTL}

	idx := &PriceIndex{
		ServiceCode: ServiceEC2,
		Region:      "us-east-1",
		UpdatedAt:   time.Now(),
		Products:    map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
	}
	if err := c.saveToCache(idx); err != nil {
		t.Fatalf("saveToCache() error: %v", err)
	}

	missing := c.Validate(map[ServiceCode][]string{
		ServiceEC2: {"us-east-1", "eu-west-1"},
	})

	if len(missing) != 1 {
		t.Fatalf("expected 1 missing entry, got %d", len(missing))
	}
	if missing[0].Service != ServiceEC2 || missing[0].Region != "eu-west-1" {
		t.Errorf("unexpected missing entry: %+v", missing[0])
	}
}

func TestCleanExpired(t *testing.T) {
	tmpDir := t.TempDir()
	ttl := 1 * time.Hour
	c := &Cache{dir: tmpDir, ttl: ttl}

	// Save a fresh index
	fresh := &PriceIndex{
		ServiceCode: ServiceEC2,
		Region:      "us-east-1",
		UpdatedAt:   time.Now(),
		Products:    map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
	}
	if err := c.saveToCache(fresh); err != nil {
		t.Fatalf("saveToCache(fresh) error: %v", err)
	}

	// Save an old index
	old := &PriceIndex{
		ServiceCode: ServiceRDS,
		Region:      "eu-west-1",
		UpdatedAt:   time.Now(),
		Products:    map[string]Price{"sku1": {SKU: "sku1", OnDemandUSD: 0.01}},
	}
	if err := c.saveToCache(old); err != nil {
		t.Fatalf("saveToCache(old) error: %v", err)
	}

	// Set old file modification time to 2 hours ago (beyond TTL)
	oldPath := c.cachePath(ServiceRDS, "eu-west-1")
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes() error: %v", err)
	}

	if err := c.CleanExpired(); err != nil {
		t.Fatalf("CleanExpired() error: %v", err)
	}

	// Fresh file should still exist
	freshPath := c.cachePath(ServiceEC2, "us-east-1")
	if _, err := os.Stat(freshPath); err != nil {
		t.Errorf("expected fresh cache file to exist, got error: %v", err)
	}

	// Old file should be removed
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("expected old cache file to be removed, stat error: %v", err)
	}
}

// newTestServer returns an httptest server that serves valid AWS pricing JSON
func newTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{
			"formatVersion": "v1.0",
			"offerCode": "AmazonEC2",
			"version": "test",
			"products": {
				"SKU1": {
					"sku": "SKU1",
					"productFamily": "Compute Instance",
					"attributes": {"instanceType": "t3.micro", "location": "US East (N. Virginia)"}
				}
			},
			"terms": {
				"OnDemand": {
					"SKU1": {
						"SKU1.T1": {
							"offerTermCode": "JRTCKXETXF",
							"sku": "SKU1",
							"priceDimensions": {
								"SKU1.T1.D1": {
									"unit": "Hrs",
									"pricePerUnit": {"USD": "0.0104"}
								}
							}
						}
					}
				}
			}
		}`)
	}))
}

func TestGetIndex_CacheHit(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewCache(tmpDir, time.Hour)

	// Pre-populate cache
	idx := &PriceIndex{
		ServiceCode: ServiceEC2,
		Region:      "us-east-1",
		Version:     "cached",
		UpdatedAt:   time.Now(),
		Products:    map[string]Price{"SKU1": {SKU: "SKU1", OnDemandUSD: 0.01}},
	}
	if err := c.saveToCache(idx); err != nil {
		t.Fatalf("saveToCache: %v", err)
	}

	got, err := c.GetIndex(context.Background(), ServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("GetIndex: %v", err)
	}
	if got.Version != "cached" {
		t.Errorf("expected cached version, got %q", got.Version)
	}
}

func TestGetIndex_CacheMiss(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	tmpDir := t.TempDir()
	c := NewCache(tmpDir, time.Hour)
	c.fetcher = &Fetcher{Client: ts.Client(), BaseURL: ts.URL}

	got, err := c.GetIndex(context.Background(), ServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("GetIndex: %v", err)
	}
	if got.Version != "test" {
		t.Errorf("expected fetched version 'test', got %q", got.Version)
	}

	// Verify it was saved to cache
	loaded, err := c.loadFromCache(ServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("expected cache to be populated: %v", err)
	}
	if loaded.Version != "test" {
		t.Errorf("cached version = %q, want 'test'", loaded.Version)
	}
}

func TestGetIndex_FetchError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	c := NewCache(tmpDir, time.Hour)
	c.fetcher = &Fetcher{Client: ts.Client(), BaseURL: ts.URL}

	_, err := c.GetIndex(context.Background(), ServiceEC2, "us-east-1")
	if err == nil {
		t.Error("expected error on fetch failure")
	}
}

func TestPrewarmCache(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	tmpDir := t.TempDir()
	c := NewCache(tmpDir, time.Hour)
	c.fetcher = &Fetcher{Client: ts.Client(), BaseURL: ts.URL}

	services := map[ServiceCode][]string{
		ServiceEC2: {"us-east-1"},
	}
	if err := c.PrewarmCache(context.Background(), services); err != nil {
		t.Fatalf("PrewarmCache: %v", err)
	}

	// Verify cached
	_, err := c.loadFromCache(ServiceEC2, "us-east-1")
	if err != nil {
		t.Errorf("expected cache to be populated after prewarm: %v", err)
	}
}

func TestPrewarmCache_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	c := NewCache(tmpDir, time.Hour)
	c.fetcher = &Fetcher{Client: ts.Client(), BaseURL: ts.URL}

	err := c.PrewarmCache(context.Background(), map[ServiceCode][]string{
		ServiceEC2: {"us-east-1"},
	})
	if err == nil {
		t.Error("expected error on prewarm failure")
	}
}
