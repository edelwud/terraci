package enginetest

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/engine"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/diskblob"
)

// MultiServicePricingServer routes pricing requests by service code.
func MultiServicePricingServer(tb testing.TB) *httptest.Server {
	tb.Helper()

	ec2PricingJSON := LoadPricingFixture(tb, "ec2_pricing")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 {
			http.NotFound(w, r)
			return
		}
		service := parts[4]

		switch service {
		case "AmazonEC2":
			fmt.Fprint(w, ec2PricingJSON)
		default:
			http.NotFound(w, r)
		}
	}))

	tb.Cleanup(ts.Close)
	return ts
}

// NewTestEstimator creates an estimator backed by a multi-service test pricing server.
func NewTestEstimator(tb testing.TB) *engine.Estimator {
	tb.Helper()

	ts := MultiServicePricingServer(tb)
	cacheDir := filepath.Join(tb.TempDir(), "cache")
	fetcher := &awskit.Fetcher{
		Client:  ts.Client(),
		BaseURL: ts.URL,
	}

	cfg := &model.CostConfig{
		Providers: model.CostProvidersConfig{"aws": {Enabled: true}},
	}
	cache := blobcache.New(diskblob.NewStore(cacheDir), model.DefaultBlobCacheNamespace, cfg.CacheTTLDuration())
	e, err := engine.NewEstimatorFromConfig(cfg, cache)
	if err != nil {
		tb.Fatalf("NewTestEstimator: %v", err)
	}
	e.SetFetcherForProvider(awskit.ProviderID, fetcher)
	return e
}

// WritePlan writes a plan.json file to the given directory.
func WritePlan(tb testing.TB, dir, planJSON string) {
	tb.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte(planJSON), 0o600); err != nil {
		tb.Fatal(err)
	}
}

// FindResource returns the ResourceCost with the given address, or nil.
func FindResource(resources []model.ResourceCost, address string) *model.ResourceCost {
	for i := range resources {
		if resources[i].Address == address {
			return &resources[i]
		}
	}
	return nil
}

// AssertCostNear asserts got is within tolerance of want.
func AssertCostNear(tb testing.TB, label string, got, want, tolerance float64) {
	tb.Helper()
	if math.Abs(got-want) > tolerance {
		tb.Errorf("%s = %.4f, want ~%.4f (tolerance %.4f)", label, got, want, tolerance)
	}
}

// LoadPlanFixture reads a plan fixture from testdata/plans/<name>.json.
func LoadPlanFixture(tb testing.TB, name string) string {
	tb.Helper()
	return string(readFixture(tb, filepath.Join("testdata", "plans", name+".json")))
}

// LoadPricingFixture reads a pricing fixture from testdata/pricing/<name>.json.
func LoadPricingFixture(tb testing.TB, name string) string {
	tb.Helper()
	return string(readFixture(tb, filepath.Join("testdata", "pricing", name+".json")))
}

// AssertUnsupportedResource verifies the common unsupported-resource contract.
func AssertUnsupportedResource(tb testing.TB, resources []model.ResourceCost, address, wantProvider string, wantKind model.CostErrorKind) *model.ResourceCost {
	tb.Helper()

	rc := FindResource(resources, address)
	if rc == nil {
		tb.Fatalf("missing resource %q in results", address)
		return nil
	}
	if rc.Provider != wantProvider {
		tb.Fatalf("Provider = %q, want %q", rc.Provider, wantProvider)
	}
	if rc.ErrorKind != wantKind {
		tb.Fatalf("ErrorKind = %q, want %q", rc.ErrorKind, wantKind)
	}
	return rc
}

// AssertUsageBasedResource verifies the common usage-based resource contract.
func AssertUsageBasedResource(tb testing.TB, resources []model.ResourceCost, address string) *model.ResourceCost {
	tb.Helper()

	rc := FindResource(resources, address)
	if rc == nil {
		tb.Fatalf("missing resource %q in results", address)
		return nil
	}
	if rc.ErrorKind != model.CostErrorUsageBased {
		tb.Fatalf("ErrorKind = %q, want %q", rc.ErrorKind, model.CostErrorUsageBased)
	}
	if rc.PriceSource != "usage-based" {
		tb.Fatalf("PriceSource = %q, want %q", rc.PriceSource, "usage-based")
	}
	return rc
}

// AssertFixedResource verifies the common fixed-cost resource contract.
func AssertFixedResource(tb testing.TB, resources []model.ResourceCost, address string, wantMonthly, tolerance float64) *model.ResourceCost {
	tb.Helper()

	rc := FindResource(resources, address)
	if rc == nil {
		tb.Fatalf("missing resource %q in results", address)
		return nil
	}
	AssertCostNear(tb, address+" monthly", rc.MonthlyCost, wantMonthly, tolerance)
	if rc.PriceSource != "fixed" {
		tb.Fatalf("PriceSource = %q, want %q", rc.PriceSource, "fixed")
	}
	return rc
}

func readFixture(tb testing.TB, relPath string) []byte {
	tb.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("failed to resolve enginetest fixture path")
	}

	path := filepath.Join(filepath.Dir(filename), relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read fixture %q: %v", relPath, err)
	}
	return data
}
