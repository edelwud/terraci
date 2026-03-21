package cost

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
	"github.com/edelwud/terraci/internal/terraform/plan"
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
	if len(result.Resources) != 1 {
		t.Errorf("resources = %d, want 1", len(result.Resources))
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

func TestGetResourceAttrs(t *testing.T) {
	tests := []struct {
		name     string
		rc       plan.ResourceChange
		wantKeys []string
	}{
		{"extracts new", plan.ResourceChange{Attributes: []plan.AttrDiff{{Path: "a", NewValue: "1"}, {Path: "b", NewValue: "2"}}}, []string{"a", "b"}},
		{"fallback old", plan.ResourceChange{Attributes: []plan.AttrDiff{{Path: "a", OldValue: "1"}}}, []string{"a"}},
		{"skip computed", plan.ResourceChange{Attributes: []plan.AttrDiff{{Path: "id", NewValue: "(known after apply)"}, {Path: "a", NewValue: "1"}}}, []string{"a"}},
		{"empty", plan.ResourceChange{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getResourceAttrs(tt.rc)
			for _, k := range tt.wantKeys {
				if _, ok := got[k]; !ok {
					t.Errorf("missing %q", k)
				}
			}
			if tt.wantKeys == nil && len(got) != 0 {
				t.Errorf("expected empty, got %v", got)
			}
		})
	}
}

func TestParseStateResources(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantKey string
	}{
		{"basic", `{"resources":[{"type":"aws_instance","name":"web","instances":[{"attributes":{}}]}]}`, 1, "aws_instance.web"},
		{"module", `{"resources":[{"type":"aws_vpc","name":"m","module":"module.n","instances":[{"attributes":{}}]}]}`, 1, "module.n.aws_vpc.m"},
		{"invalid", "x", 0, ""},
		{"empty", `{"resources":[]}`, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStateResources([]byte(tt.input))
			if len(got) != tt.wantLen {
				t.Errorf("len=%d want %d", len(got), tt.wantLen)
			}
			if tt.wantKey != "" {
				if _, ok := got[tt.wantKey]; !ok {
					t.Errorf("missing %q", tt.wantKey)
				}
			}
		})
	}
}

func TestBuildResourceAddress(t *testing.T) {
	tests := []struct {
		module, typ, name string
		idx               any
		want              string
	}{
		{"", "aws_instance", "web", nil, "aws_instance.web"},
		{"module.vpc", "aws_vpc", "main", nil, "module.vpc.aws_vpc.main"},
		{"", "aws_instance", "web", "foo", `aws_instance.web["foo"]`},
		{"", "aws_instance", "web", float64(0), "aws_instance.web[0]"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := buildResourceAddress(tt.module, tt.typ, tt.name, tt.idx); got != tt.want {
				t.Errorf("got %q", got)
			}
		})
	}
}
