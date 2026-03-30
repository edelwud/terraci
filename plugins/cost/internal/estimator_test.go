package costengine

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	_ "github.com/edelwud/terraci/plugins/cost/internal/cloud/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// multiServicePricingServer returns an httptest server that routes requests by
// service code extracted from the URL path. Serves EC2 (t3.micro + gp2 storage)
// for all EC2 requests, and returns 404 for unknown services.
func multiServicePricingServer(t *testing.T) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// URL pattern: /offers/v1.0/aws/{service}/current/{region}/index.json
		parts := strings.Split(r.URL.Path, "/")
		// parts: ["", "offers", "v1.0", "aws", "{service}", "current", "{region}", "index.json"]
		if len(parts) < 5 {
			http.NotFound(w, r)
			return
		}
		service := parts[4]

		switch service {
		case "AmazonEC2":
			fmt.Fprint(w, ec2PricingJSON)
		default:
			// Unknown services get 404 — forces handlers to use fallback pricing
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

// ec2PricingJSON serves t3.micro compute ($0.0104/hr) and gp2 storage ($0.10/GB-Mo).
const ec2PricingJSON = `{
	"formatVersion": "v1.0",
	"offerCode": "AmazonEC2",
	"version": "test-v1",
	"products": {
		"SKU_T3MICRO": {
			"sku": "SKU_T3MICRO",
			"productFamily": "Compute Instance",
			"attributes": {
				"instanceType": "t3.micro",
				"location": "US East (N. Virginia)",
				"tenancy": "Shared",
				"operatingSystem": "Linux",
				"preInstalledSw": "NA",
				"capacitystatus": "Used"
			}
		},
		"SKU_GP2": {
			"sku": "SKU_GP2",
			"productFamily": "Storage",
			"attributes": {
				"volumeApiName": "gp2",
				"location": "US East (N. Virginia)"
			}
		}
	},
	"terms": {
		"OnDemand": {
			"SKU_T3MICRO": {
				"SKU_T3MICRO.T1": {
					"offerTermCode": "JRTCKXETXF",
					"sku": "SKU_T3MICRO",
					"priceDimensions": {
						"SKU_T3MICRO.T1.D1": {
							"unit": "Hrs",
							"pricePerUnit": {"USD": "0.0104"}
						}
					}
				}
			},
			"SKU_GP2": {
				"SKU_GP2.T1": {
					"offerTermCode": "JRTCKXETXF",
					"sku": "SKU_GP2",
					"priceDimensions": {
						"SKU_GP2.T1.D1": {
							"unit": "GB-Mo",
							"pricePerUnit": {"USD": "0.10"}
						}
					}
				}
			}
		}
	}
}`

// newTestEstimator creates an estimator backed by a multi-service httptest server.
func newTestEstimator(t *testing.T) *Estimator {
	t.Helper()
	ts := multiServicePricingServer(t)
	cacheDir := filepath.Join(t.TempDir(), "cache")
	fetcher := &awskit.Fetcher{
		Client:  ts.Client(),
		BaseURL: ts.URL,
	}
	return NewEstimator(cacheDir, 0, fetcher)
}

// writePlan writes a plan.json file to the given directory.
func writePlan(t *testing.T, dir, planJSON string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte(planJSON), 0o600); err != nil {
		t.Fatal(err)
	}
}

// findResource returns the ResourceCost with the given address, or nil.
func findResource(resources []ResourceCost, address string) *ResourceCost {
	for i := range resources {
		if resources[i].Address == address {
			return &resources[i]
		}
	}
	return nil
}

// assertCostNear asserts got is within tolerance of want.
func assertCostNear(t *testing.T, label string, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Errorf("%s = %.4f, want ~%.4f (tolerance %.4f)", label, got, want, tolerance)
	}
}

// --- Plan JSON fixtures ---

// planCreateEC2 creates a single t3.micro instance.
const planCreateEC2 = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [{
		"address": "aws_instance.web",
		"module_address": "",
		"type": "aws_instance",
		"name": "web",
		"change": {
			"actions": ["create"],
			"before": null,
			"after": {"instance_type": "t3.micro", "ami": "ami-12345"},
			"after_unknown": {}
		}
	}]
}`

// planDeleteEC2 deletes a t3.micro instance.
const planDeleteEC2 = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [{
		"address": "aws_instance.old",
		"module_address": "",
		"type": "aws_instance",
		"name": "old",
		"change": {
			"actions": ["delete"],
			"before": {"instance_type": "t3.micro", "ami": "ami-12345"},
			"after": null,
			"after_unknown": {}
		}
	}]
}`

// planUpdateEC2 updates a t3.micro instance (tags only — same cost).
const planUpdateEC2 = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [{
		"address": "aws_instance.web",
		"module_address": "",
		"type": "aws_instance",
		"name": "web",
		"change": {
			"actions": ["update"],
			"before": {"instance_type": "t3.micro", "ami": "ami-12345"},
			"after": {"instance_type": "t3.micro", "ami": "ami-12345", "tags": {"env": "prod"}},
			"after_unknown": {}
		}
	}]
}`

// planNoOp — instance exists with no changes.
const planNoOp = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [{
		"address": "aws_instance.web",
		"module_address": "",
		"type": "aws_instance",
		"name": "web",
		"change": {
			"actions": ["no-op"],
			"before": {"instance_type": "t3.micro", "ami": "ami-12345"},
			"after": {"instance_type": "t3.micro", "ami": "ami-12345"},
			"after_unknown": {}
		}
	}]
}`

// planUnsupportedResource — a resource type with no handler.
const planUnsupportedResource = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [{
		"address": "aws_cloudfront_distribution.main",
		"module_address": "",
		"type": "aws_cloudfront_distribution",
		"name": "main",
		"change": {
			"actions": ["create"],
			"before": null,
			"after": {"enabled": true},
			"after_unknown": {}
		}
	}]
}`

// planUsageBased — SQS queue (usage-based, returns $0).
const planUsageBased = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [{
		"address": "aws_sqs_queue.main",
		"module_address": "",
		"type": "aws_sqs_queue",
		"name": "main",
		"change": {
			"actions": ["create"],
			"before": null,
			"after": {"name": "my-queue"},
			"after_unknown": {}
		}
	}]
}`

// planFixedCost — Route53 zone ($0.50/mo fixed).
const planFixedCost = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [{
		"address": "aws_route53_zone.main",
		"module_address": "",
		"type": "aws_route53_zone",
		"name": "main",
		"change": {
			"actions": ["create"],
			"before": null,
			"after": {"name": "example.com"},
			"after_unknown": {}
		}
	}]
}`

// planMixedActions — create, no-op, and unsupported resources together.
const planMixedActions = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [
		{
			"address": "aws_instance.new",
			"module_address": "",
			"type": "aws_instance",
			"name": "new",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {"instance_type": "t3.micro"},
				"after_unknown": {}
			}
		},
		{
			"address": "aws_route53_zone.dns",
			"module_address": "",
			"type": "aws_route53_zone",
			"name": "dns",
			"change": {
				"actions": ["no-op"],
				"before": {"name": "example.com"},
				"after": {"name": "example.com"},
				"after_unknown": {}
			}
		},
		{
			"address": "aws_sqs_queue.events",
			"module_address": "",
			"type": "aws_sqs_queue",
			"name": "events",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {"name": "events"},
				"after_unknown": {}
			}
		}
	]
}`

// planWithModules — resources in different Terraform modules.
const planWithModules = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [
		{
			"address": "module.vpc.aws_route53_zone.main",
			"module_address": "module.vpc",
			"type": "aws_route53_zone",
			"name": "main",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {"name": "example.com"},
				"after_unknown": {}
			}
		},
		{
			"address": "module.compute.aws_instance.web",
			"module_address": "module.compute",
			"type": "aws_instance",
			"name": "web",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {"instance_type": "t3.micro"},
				"after_unknown": {}
			}
		}
	]
}`

// planEmpty has no resource changes.
const planEmpty = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": []
}`

// --- Tests ---

func TestEstimateModule_CreateAction(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planCreateEC2)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	// t3.micro: $0.0104/hr * 730 = $7.592
	// root_block_device (8GB gp2): $0.10 * 8 = $0.80
	// Total: $8.392
	assertCostNear(t, "AfterCost", result.AfterCost, 8.392, 0.01)
	assertCostNear(t, "BeforeCost", result.BeforeCost, 0, 0.001)
	assertCostNear(t, "DiffCost", result.DiffCost, 8.392, 0.01)

	// Verify resources: instance + synthesized root_block_device
	rc := findResource(result.Resources, "aws_instance.web")
	if rc == nil {
		t.Fatal("missing aws_instance.web in resources")
	}
	assertCostNear(t, "instance monthly", rc.MonthlyCost, 7.592, 0.01)
	if rc.PriceSource != "aws-bulk-api" {
		t.Errorf("PriceSource = %q, want aws-bulk-api", rc.PriceSource)
	}
	if rc.ErrorKind != CostErrorNone {
		t.Errorf("ErrorKind = %q, want empty", rc.ErrorKind)
	}

	rootVol := findResource(result.Resources, "aws_instance.web/root_volume")
	if rootVol == nil {
		t.Fatal("missing synthesized root_volume sub-resource")
	}
	assertCostNear(t, "root_volume monthly", rootVol.MonthlyCost, 0.80, 0.01)

	if !result.HasChanges {
		t.Error("HasChanges should be true for create action")
	}
}

func TestEstimateModule_DeleteAction(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planDeleteEC2)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	// Delete: cost goes into BeforeCost (resource existed before)
	if result.BeforeCost <= 0 {
		t.Errorf("BeforeCost = %.4f, want > 0 (delete removes existing cost)", result.BeforeCost)
	}
	assertCostNear(t, "AfterCost", result.AfterCost, 0, 0.001)
	if result.DiffCost >= 0 {
		t.Errorf("DiffCost = %.4f, want < 0 (cost reduction)", result.DiffCost)
	}
}

func TestEstimateModule_UpdateAction(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planUpdateEC2)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	// Update with same instance type: before and after costs should be equal
	if result.BeforeCost <= 0 {
		t.Errorf("BeforeCost = %.4f, want > 0", result.BeforeCost)
	}
	if result.AfterCost <= 0 {
		t.Errorf("AfterCost = %.4f, want > 0", result.AfterCost)
	}

	rc := findResource(result.Resources, "aws_instance.web")
	if rc == nil {
		t.Fatal("missing aws_instance.web")
	}
	// Both before and after should have costs
	if rc.BeforeMonthlyCost <= 0 {
		t.Errorf("BeforeMonthlyCost = %.4f, want > 0", rc.BeforeMonthlyCost)
	}
	if rc.MonthlyCost <= 0 {
		t.Errorf("MonthlyCost = %.4f, want > 0", rc.MonthlyCost)
	}
}

func TestEstimateModule_NoOpAction(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planNoOp)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	// No-op: before and after costs are the same, diff is zero
	assertCostNear(t, "DiffCost", result.DiffCost, 0, 0.01)
	if result.BeforeCost <= 0 {
		t.Errorf("BeforeCost = %.4f, want > 0 (existing resource)", result.BeforeCost)
	}
	assertCostNear(t, "BeforeCost == AfterCost", result.BeforeCost, result.AfterCost, 0.01)
}

func TestEstimateModule_UnsupportedResource(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planUnsupportedResource)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	rc := findResource(result.Resources, "aws_cloudfront_distribution.main")
	if rc == nil {
		t.Fatal("missing resource in results")
	}
	if rc.ErrorKind != CostErrorNoProvider {
		t.Errorf("ErrorKind = %q, want %q", rc.ErrorKind, CostErrorNoProvider)
	}
	if rc.MonthlyCost != 0 {
		t.Errorf("MonthlyCost = %.4f, want 0 for unsupported", rc.MonthlyCost)
	}
	if result.Unsupported != 1 {
		t.Errorf("Unsupported = %d, want 1", result.Unsupported)
	}
}

func TestEstimateModule_KnownProviderMissingHandler(t *testing.T) {
	registry := handler.NewRegistry()
	router := NewResourceProviderRouter()
	router.Register(awskit.ProviderID, handler.ResourceType("aws_cloudfront_distribution"))

	ts := multiServicePricingServer(t)
	cacheDir := filepath.Join(t.TempDir(), "cache")
	fetcher := &awskit.Fetcher{
		Client:  ts.Client(),
		BaseURL: ts.URL,
	}

	awsProvider, ok := cloud.Get(awskit.ProviderID)
	if !ok {
		t.Fatal("aws provider not registered")
	}

	def := awsProvider.Definition()
	runtimes := map[string]*ProviderRuntime{
		awskit.ProviderID: {
			Definition: def,
			Cache:      pricing.NewCache(cacheDir, 0, fetcher),
		},
	}
	runtimeRegistry := NewProviderRuntimeRegistry(cloud.Providers(), runtimes)

	resolver := NewCostResolver(router, registry, nil)
	runtimeRegistry.router = router
	e := newEstimator(runtimeRegistry, resolver)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planUnsupportedResource)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	rc := findResource(result.Resources, "aws_cloudfront_distribution.main")
	if rc == nil {
		t.Fatal("missing resource in results")
	}
	if rc.Provider != awskit.ProviderID {
		t.Errorf("Provider = %q, want %q", rc.Provider, awskit.ProviderID)
	}
	if rc.ErrorKind != CostErrorNoHandler {
		t.Errorf("ErrorKind = %q, want %q", rc.ErrorKind, CostErrorNoHandler)
	}
	if rc.ErrorDetail != "no handler" {
		t.Errorf("ErrorDetail = %q, want %q", rc.ErrorDetail, "no handler")
	}
	if result.Unsupported != 1 {
		t.Errorf("Unsupported = %d, want 1", result.Unsupported)
	}
}

func TestEstimateModule_UsageBasedResource(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planUsageBased)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	rc := findResource(result.Resources, "aws_sqs_queue.main")
	if rc == nil {
		t.Fatal("missing resource in results")
	}
	if rc.ErrorKind != CostErrorUsageBased {
		t.Errorf("ErrorKind = %q, want %q", rc.ErrorKind, CostErrorUsageBased)
	}
	if rc.PriceSource != "usage-based" {
		t.Errorf("PriceSource = %q, want usage-based", rc.PriceSource)
	}
	// Usage-based should NOT increment Unsupported counter
	if result.Unsupported != 0 {
		t.Errorf("Unsupported = %d, want 0 (usage-based is not unsupported)", result.Unsupported)
	}
}

func TestEstimateModule_FixedCostResource(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planFixedCost)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	rc := findResource(result.Resources, "aws_route53_zone.main")
	if rc == nil {
		t.Fatal("missing resource in results")
	}
	assertCostNear(t, "Route53 monthly", rc.MonthlyCost, 0.50, 0.01)
	if rc.PriceSource != "fixed" {
		t.Errorf("PriceSource = %q, want fixed", rc.PriceSource)
	}
	assertCostNear(t, "AfterCost", result.AfterCost, 0.50, 0.01)
}

func TestEstimateModule_MixedActions(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planMixedActions)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	// aws_instance.new (create): ~$7.59 + ~$0.80 root vol = ~$8.39
	// aws_route53_zone.dns (no-op): $0.50 in before AND after
	// aws_sqs_queue.events (create, usage-based): $0
	// Expected: AfterCost ≈ $8.39 + $0.50 = $8.89, BeforeCost ≈ $0.50
	assertCostNear(t, "AfterCost", result.AfterCost, 8.892, 0.05)
	assertCostNear(t, "BeforeCost", result.BeforeCost, 0.50, 0.01)

	// Verify resource count: instance + root_vol + route53 + sqs = 4
	if len(result.Resources) != 4 {
		t.Errorf("resources count = %d, want 4", len(result.Resources))
	}
}

func TestEstimateModule_CompoundResource(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planCreateEC2)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	// EC2 instance should synthesize root_block_device as sub-resource
	var found bool
	for _, rc := range result.Resources {
		if strings.Contains(rc.Address, "root_volume") {
			found = true
			if rc.Type != "aws_ebs_volume" {
				t.Errorf("synthesized type = %q, want aws_ebs_volume", rc.Type)
			}
			assertCostNear(t, "root_volume cost", rc.MonthlyCost, 0.80, 0.01) // 8GB * $0.10
			break
		}
	}
	if !found {
		t.Error("CompoundHandler did not synthesize root_block_device")
	}
}

func TestEstimateModule_SubmoduleGrouping(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planWithModules)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	if len(result.Submodules) < 2 {
		t.Fatalf("submodules = %d, want >= 2 (module.vpc, module.compute)", len(result.Submodules))
	}

	// Verify both modules are present
	var foundVPC, foundCompute bool
	for _, sm := range result.Submodules {
		switch sm.ModuleAddr {
		case "module.vpc":
			foundVPC = true
			assertCostNear(t, "vpc cost", sm.MonthlyCost, 0.50, 0.01)
		case "module.compute":
			foundCompute = true
			if sm.MonthlyCost <= 0 {
				t.Errorf("compute cost = %.4f, want > 0", sm.MonthlyCost)
			}
		}
	}
	if !foundVPC {
		t.Error("missing module.vpc in submodules")
	}
	if !foundCompute {
		t.Error("missing module.compute in submodules")
	}
}

func TestEstimateModule_EmptyPlan(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planEmpty)

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	if len(result.Resources) != 0 {
		t.Errorf("resources = %d, want 0", len(result.Resources))
	}
	assertCostNear(t, "AfterCost", result.AfterCost, 0, 0.001)
	assertCostNear(t, "BeforeCost", result.BeforeCost, 0, 0.001)
	if result.HasChanges {
		t.Error("HasChanges should be false for empty plan")
	}
}

func TestEstimateModule_InvalidPlanJSON(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, `{"invalid": "json"`)

	_, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err == nil {
		t.Fatal("expected error for invalid plan.json")
	}
}

func TestEstimateModule_NoPlanFile(t *testing.T) {
	e := newTestEstimator(t)
	dir := t.TempDir()

	_, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err == nil {
		t.Fatal("expected error when plan.json doesn't exist")
	}
}

func TestEstimateModules_Concurrent(t *testing.T) {
	e := newTestEstimator(t)
	tmpDir := t.TempDir()

	modules := []string{"vpc", "rds", "eks"}
	paths := make([]string, len(modules))
	regions := make(map[string]string)

	for i, mod := range modules {
		dir := filepath.Join(tmpDir, "platform", "prod", "us-east-1", mod)
		writePlan(t, dir, planCreateEC2)
		paths[i] = dir
		regions[dir] = "us-east-1"
	}

	result, err := e.EstimateModules(context.Background(), paths, regions)
	if err != nil {
		t.Fatalf("EstimateModules: %v", err)
	}

	if len(result.Modules) != 3 {
		t.Fatalf("modules = %d, want 3", len(result.Modules))
	}

	// Each module has same plan, so costs should be identical
	for i, mc := range result.Modules {
		if mc.AfterCost <= 0 {
			t.Errorf("module[%d].AfterCost = %.4f, want > 0", i, mc.AfterCost)
		}
		if mc.Error != "" {
			t.Errorf("module[%d].Error = %q, want empty", i, mc.Error)
		}
	}

	// Total should be 3x single module cost
	assertCostNear(t, "TotalAfter", result.TotalAfter, result.Modules[0].AfterCost*3, 0.01)
	if result.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", result.Currency)
	}
	if result.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
}

func TestEstimateModules_PartialFailure(t *testing.T) {
	e := newTestEstimator(t)
	tmpDir := t.TempDir()

	// One valid module, one with invalid plan
	validDir := filepath.Join(tmpDir, "valid")
	writePlan(t, validDir, planCreateEC2)

	invalidDir := filepath.Join(tmpDir, "invalid")
	writePlan(t, invalidDir, `not json at all`)

	result, err := e.EstimateModules(context.Background(),
		[]string{validDir, invalidDir},
		map[string]string{validDir: "us-east-1", invalidDir: "us-east-1"})
	if err != nil {
		t.Fatalf("EstimateModules should not return error for partial failure: %v", err)
	}

	if len(result.Modules) != 2 {
		t.Fatalf("modules = %d, want 2", len(result.Modules))
	}

	// One should have error, one should be fine
	var hasError, hasSuccess bool
	for _, mc := range result.Modules {
		if mc.Error != "" {
			hasError = true
		} else if mc.AfterCost > 0 {
			hasSuccess = true
		}
	}
	if !hasError {
		t.Error("expected one module with error")
	}
	if !hasSuccess {
		t.Error("expected one module with success")
	}
}

func TestEstimateModules_DefaultRegion(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planCreateEC2)

	// Pass empty regions map — should use DefaultRegion
	result, err := e.EstimateModules(context.Background(), []string{dir}, map[string]string{})
	if err != nil {
		t.Fatalf("EstimateModules: %v", err)
	}

	if len(result.Modules) != 1 {
		t.Fatalf("modules = %d, want 1", len(result.Modules))
	}
	if result.Modules[0].Region != DefaultRegion {
		t.Errorf("Region = %q, want %q", result.Modules[0].Region, DefaultRegion)
	}
}

func TestValidateAndPrefetch_DownloadsMissing(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planCreateEC2)

	regions := map[string]string{dir: "us-east-1"}
	err := e.ValidateAndPrefetch(context.Background(), []string{dir}, regions)
	if err != nil {
		t.Fatalf("ValidateAndPrefetch: %v", err)
	}

	// After prefetch, cache should have EC2 pricing
	entries := e.CacheEntries()
	if len(entries) == 0 {
		t.Error("cache should have entries after prefetch")
	}
}

func TestValidateAndPrefetch_SkipsUsageBased(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planUsageBased)

	regions := map[string]string{dir: "us-east-1"}
	err := e.ValidateAndPrefetch(context.Background(), []string{dir}, regions)
	if err != nil {
		t.Fatalf("ValidateAndPrefetch: %v", err)
	}

	// Usage-based resources should not trigger any downloads
	entries := e.CacheEntries()
	if len(entries) != 0 {
		t.Errorf("cache entries = %d, want 0 (no standard resources to fetch)", len(entries))
	}
}

func TestValidateAndPrefetch_SkipsCachedData(t *testing.T) {
	e := newTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	writePlan(t, dir, planCreateEC2)

	regions := map[string]string{dir: "us-east-1"}

	// First call: downloads
	if err := e.ValidateAndPrefetch(context.Background(), []string{dir}, regions); err != nil {
		t.Fatalf("first prefetch: %v", err)
	}

	// Second call: should succeed without downloading (data cached)
	if err := e.ValidateAndPrefetch(context.Background(), []string{dir}, regions); err != nil {
		t.Fatalf("second prefetch: %v", err)
	}
}

func TestNewEstimator(t *testing.T) {
	e := NewEstimator("", 0, awskit.NewFetcher())
	if e == nil {
		t.Fatal("NewEstimator returned nil")
	}
}

func TestNewEstimatorFromConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		e, err := NewEstimatorFromConfig(nil)
		if err != nil {
			t.Fatalf("NewEstimatorFromConfig() error = %v", err)
		}
		if e == nil {
			t.Fatal("expected non-nil estimator")
		}
	})

	t.Run("custom cache dir and TTL", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &CostConfig{
			CacheDir: tmpDir,
			CacheTTL: "2h",
			Providers: CostProvidersConfig{
				AWS: &ProviderConfig{Enabled: true},
			},
		}
		e, err := NewEstimatorFromConfig(cfg)
		if err != nil {
			t.Fatalf("NewEstimatorFromConfig() error = %v", err)
		}
		if e.CacheDir() != tmpDir {
			t.Errorf("CacheDir() = %q, want %q", e.CacheDir(), tmpDir)
		}
		if e.CacheTTL() != 2*time.Hour {
			t.Errorf("CacheTTL() = %v, want 2h", e.CacheTTL())
		}
	})

	t.Run("invalid TTL uses default", func(t *testing.T) {
		cfg := &CostConfig{
			CacheTTL: "invalid",
			Providers: CostProvidersConfig{
				AWS: &ProviderConfig{Enabled: true},
			},
		}
		e, err := NewEstimatorFromConfig(cfg)
		if err != nil {
			t.Fatalf("NewEstimatorFromConfig() error = %v", err)
		}
		if e.CacheTTL() != 24*time.Hour {
			t.Errorf("CacheTTL() = %v, want 24h", e.CacheTTL())
		}
	})
}

func TestEstimator_CacheAccessors(t *testing.T) {
	cacheDir := t.TempDir()
	e := NewEstimator(cacheDir, 0, awskit.NewFetcher())

	if e.CacheDir() != cacheDir {
		t.Errorf("CacheDir() = %q, want %q", e.CacheDir(), cacheDir)
	}
	if e.CacheTTL() <= 0 {
		t.Errorf("CacheTTL() = %v, want > 0", e.CacheTTL())
	}
	if e.CacheOldestAge() != 0 {
		t.Errorf("CacheOldestAge() = %v, want 0 for empty cache", e.CacheOldestAge())
	}
	if len(e.CacheEntries()) != 0 {
		t.Errorf("CacheEntries() len = %d, want 0", len(e.CacheEntries()))
	}
	e.CleanExpiredCache() // should not panic
}

func TestAggregateCost(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		cost       float64
		wantBefore float64
		wantAfter  float64
	}{
		{"create", "create", 10, 0, 10},
		{"delete", "delete", 10, 10, 0},
		{"update", "update", 10, 10, 10},
		{"replace", "replace", 10, 10, 10},
		{"no-op", "no-op", 10, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ModuleCost{}
			rc := ResourceCost{MonthlyCost: tt.cost, BeforeMonthlyCost: tt.cost}
			aggregateCost(result, rc, tt.action)

			assertCostNear(t, "BeforeCost", result.BeforeCost, tt.wantBefore, 0.001)
			assertCostNear(t, "AfterCost", result.AfterCost, tt.wantAfter, 0.001)
		})
	}
}

func TestAggregateCost_UnsupportedNotCounted(t *testing.T) {
	result := &ModuleCost{}
	rc := ResourceCost{
		MonthlyCost: 100,
		ErrorKind:   CostErrorNoHandler,
	}
	aggregateCost(result, rc, "create")

	if result.AfterCost != 0 {
		t.Errorf("AfterCost = %.2f, want 0 (unsupported should not add cost)", result.AfterCost)
	}
	if result.Unsupported != 1 {
		t.Errorf("Unsupported = %d, want 1", result.Unsupported)
	}
}
