package cost

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/edelwud/terraci/internal/cost/pricing"
	"github.com/edelwud/terraci/pkg/config"
)

// fakePricingServer returns an httptest server serving minimal AWS pricing JSON.
func fakePricingServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{
			"formatVersion": "v1.0",
			"offerCode": "AmazonEC2",
			"version": "test",
			"products": {
				"SKU1": {
					"sku": "SKU1",
					"productFamily": "Compute Instance",
					"attributes": {
						"instanceType": "t3.micro",
						"location": "US East (N. Virginia)",
						"tenancy": "Shared",
						"operatingSystem": "Linux",
						"preInstalledSw": "NA",
						"capacitystatus": "Used"
					}
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

// newTestEstimator creates an estimator backed by httptest instead of real AWS.
func newTestEstimator(t *testing.T) (estimator *Estimator, cleanup func()) {
	t.Helper()
	ts := fakePricingServer()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	estimator = NewEstimator(cacheDir, 0)
	estimator.SetPricingFetcher(&pricing.Fetcher{
		Client:  ts.Client(),
		BaseURL: ts.URL,
	})
	return estimator, ts.Close
}

func TestEstimator_EstimateModule(t *testing.T) {
	estimator, cleanup := newTestEstimator(t)
	defer cleanup()

	modulePath := filepath.Join(t.TempDir(), "platform", "prod", "us-east-1", "vpc")
	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(modulePath, "plan.json"), []byte(`{
		"format_version": "1.2",
		"terraform_version": "1.6.0",
		"resource_changes": [{
			"address": "aws_instance.web",
			"type": "aws_instance",
			"name": "web",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {"instance_type": "t3.micro", "ami": "ami-12345"}
			}
		}]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := estimator.EstimateModule(context.Background(), modulePath, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}
	// 1 instance + 1 synthesized root_volume sub-resource
	if len(result.Resources) < 1 {
		t.Errorf("resources = %d, want >= 1", len(result.Resources))
	}
	if result.AfterCost <= 0 {
		t.Errorf("AfterCost = %.4f, want > 0", result.AfterCost)
	}
	t.Logf("t3.micro cost: $%.2f/month", result.AfterCost)
}

func TestEstimator_ValidateAndPrefetch(t *testing.T) {
	estimator, cleanup := newTestEstimator(t)
	defer cleanup()

	modulePath := filepath.Join(t.TempDir(), "test", "module")
	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulePath, "plan.json"), []byte(`{
		"format_version": "1.2",
		"resource_changes": [{
			"address": "aws_instance.web",
			"type": "aws_instance",
			"name": "web",
			"change": {"actions": ["create"], "before": null, "after": {"instance_type": "t3.micro"}}
		}]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}

	err := estimator.ValidateAndPrefetch(context.Background(),
		[]string{modulePath}, map[string]string{modulePath: "us-east-1"})
	if err != nil {
		t.Fatalf("ValidateAndPrefetch: %v", err)
	}
}

func TestEstimator_EstimateModules(t *testing.T) {
	estimator, cleanup := newTestEstimator(t)
	defer cleanup()

	tmpDir := t.TempDir()
	modules := []string{"vpc", "eks"}
	modulePaths := make([]string, 0, len(modules))
	regions := make(map[string]string)

	for _, mod := range modules {
		modPath := filepath.Join(tmpDir, "platform", "prod", "us-east-1", mod)
		if err := os.MkdirAll(modPath, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(modPath, "plan.json"), []byte(`{
			"format_version": "1.2",
			"resource_changes": [{
				"address": "aws_instance.main",
				"type": "aws_instance",
				"name": "main",
				"change": {"actions": ["create"], "before": null, "after": {"instance_type": "t3.micro"}}
			}]
		}`), 0o600); err != nil {
			t.Fatal(err)
		}
		modulePaths = append(modulePaths, modPath)
		regions[modPath] = "us-east-1"
	}

	result, err := estimator.EstimateModules(context.Background(), modulePaths, regions)
	if err != nil {
		t.Fatalf("EstimateModules: %v", err)
	}

	if len(result.Modules) != 2 {
		t.Errorf("modules = %d, want 2", len(result.Modules))
	}
	if result.TotalAfter <= 0 {
		t.Errorf("TotalAfter = %.2f, want > 0", result.TotalAfter)
	}
	t.Logf("total: $%.2f/month", result.TotalAfter)
}

func TestNewEstimator(t *testing.T) {
	estimator := NewEstimator("", 0)
	if estimator == nil {
		t.Fatal("nil")
	}
}

func TestNewEstimatorFromConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		e := NewEstimatorFromConfig(nil)
		if e == nil {
			t.Fatal("expected non-nil estimator")
		}
	})

	t.Run("with custom cache dir and TTL", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.CostConfig{
			CacheDir: tmpDir,
			CacheTTL: "2h",
		}
		e := NewEstimatorFromConfig(cfg)
		if e.CacheDir() != tmpDir {
			t.Errorf("CacheDir() = %q, want %q", e.CacheDir(), tmpDir)
		}
		if e.CacheTTL() != 2*time.Hour {
			t.Errorf("CacheTTL() = %v, want 2h", e.CacheTTL())
		}
	})

	t.Run("invalid TTL uses default", func(t *testing.T) {
		cfg := &config.CostConfig{CacheTTL: "invalid"}
		e := NewEstimatorFromConfig(cfg)
		if e.CacheTTL() != 24*time.Hour {
			t.Errorf("CacheTTL() = %v, want 24h", e.CacheTTL())
		}
	})
}

func TestEstimator_Accessors(t *testing.T) {
	cacheDir := t.TempDir()
	estimator := NewEstimator(cacheDir, 0)

	t.Run("CacheDir", func(t *testing.T) {
		got := estimator.CacheDir()
		if got != cacheDir {
			t.Errorf("CacheDir() = %q, want %q", got, cacheDir)
		}
	})

	t.Run("CacheTTL", func(t *testing.T) {
		ttl := estimator.CacheTTL()
		if ttl <= 0 {
			t.Errorf("CacheTTL() = %v, want > 0", ttl)
		}
	})

	t.Run("CacheOldestAge empty", func(t *testing.T) {
		if estimator.CacheOldestAge() != 0 {
			t.Errorf("CacheOldestAge() = %v, want 0", estimator.CacheOldestAge())
		}
	})

	t.Run("CacheEntries empty", func(t *testing.T) {
		entries := estimator.CacheEntries()
		if len(entries) != 0 {
			t.Errorf("CacheEntries() len = %d, want 0", len(entries))
		}
	})

	t.Run("CleanExpiredCache no-op", func(_ *testing.T) {
		estimator.CleanExpiredCache() // should not panic
	})
}
