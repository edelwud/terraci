package engine_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	_ "github.com/edelwud/terraci/plugins/cost/internal/cloud/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/engine"
	"github.com/edelwud/terraci/plugins/cost/internal/enginetest"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
	"github.com/edelwud/terraci/plugins/diskblob"
)

// planReplaceEC2 replaces a t3.micro instance.
const planReplaceEC2 = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [{
		"address": "aws_instance.web",
		"module_address": "",
		"type": "aws_instance",
		"name": "web",
		"change": {
			"actions": ["delete", "create"],
			"before": {"instance_type": "t3.micro", "ami": "ami-12345"},
			"after": {"instance_type": "t3.micro", "ami": "ami-67890"},
			"after_unknown": {}
		}
	}]
}`

// --- Tests ---

func TestEstimateModule_CreateAction(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "create_ec2"))

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	// t3.micro: $0.0104/hr * 730 = $7.592
	// root_block_device (8GB gp2): $0.10 * 8 = $0.80
	// Total: $8.392
	enginetest.AssertCostNear(t, "AfterCost", result.AfterCost, 8.392, 0.01)
	enginetest.AssertCostNear(t, "BeforeCost", result.BeforeCost, 0, 0.001)
	enginetest.AssertCostNear(t, "DiffCost", result.DiffCost, 8.392, 0.01)

	// Verify resources: instance + synthesized root_block_device
	rc := enginetest.FindResource(result.Resources, "aws_instance.web")
	if rc == nil {
		t.Fatal("missing aws_instance.web in resources")
	}
	enginetest.AssertCostNear(t, "instance monthly", rc.MonthlyCost, 7.592, 0.01)
	if rc.PriceSource != "aws-bulk-api" {
		t.Errorf("PriceSource = %q, want aws-bulk-api", rc.PriceSource)
	}
	if rc.ErrorKind != model.CostErrorNone {
		t.Errorf("ErrorKind = %q, want empty", rc.ErrorKind)
	}

	rootVol := enginetest.FindResource(result.Resources, "aws_instance.web/root_volume")
	if rootVol == nil {
		t.Fatal("missing synthesized root_volume sub-resource")
	}
	enginetest.AssertCostNear(t, "root_volume monthly", rootVol.MonthlyCost, 0.80, 0.01)

	if !result.HasChanges {
		t.Error("HasChanges should be true for create action")
	}
}

func TestEstimateModule_DeleteAction(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "delete_ec2"))

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	// Delete: cost goes into BeforeCost (resource existed before)
	if result.BeforeCost <= 0 {
		t.Errorf("BeforeCost = %.4f, want > 0 (delete removes existing cost)", result.BeforeCost)
	}
	enginetest.AssertCostNear(t, "AfterCost", result.AfterCost, 0, 0.001)
	if result.DiffCost >= 0 {
		t.Errorf("DiffCost = %.4f, want < 0 (cost reduction)", result.DiffCost)
	}
}

func TestEstimateModule_UpdateAction(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "update_ec2"))

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

	rc := enginetest.FindResource(result.Resources, "aws_instance.web")
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
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "no_op"))

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	// No-op: before and after costs are the same, diff is zero
	enginetest.AssertCostNear(t, "DiffCost", result.DiffCost, 0, 0.01)
	if result.BeforeCost <= 0 {
		t.Errorf("BeforeCost = %.4f, want > 0 (existing resource)", result.BeforeCost)
	}
	enginetest.AssertCostNear(t, "BeforeCost == AfterCost", result.BeforeCost, result.AfterCost, 0.01)
}

func TestEstimateModule_UnsupportedResource(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "unsupported_resource"))

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	rc := enginetest.AssertUnsupportedResource(t, result.Resources, "aws_cloudfront_distribution.main", "", model.CostErrorNoProvider)
	if rc.MonthlyCost != 0 {
		t.Errorf("MonthlyCost = %.4f, want 0 for unsupported", rc.MonthlyCost)
	}
	if result.Unsupported != 1 {
		t.Errorf("Unsupported = %d, want 1", result.Unsupported)
	}
}

func TestEstimateModule_KnownProviderMissingHandler(t *testing.T) {
	registry := handler.NewRegistry()
	router := costruntime.NewResourceProviderRouter()
	router.Register(awskit.ProviderID, handler.ResourceType("aws_cloudfront_distribution"))

	ts := enginetest.MultiServicePricingServer(t)
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
	runtimes := map[string]*costruntime.ProviderRuntime{
		awskit.ProviderID: {
			Definition: def,
			Cache:      pricing.NewCache(diskblob.NewStore(cacheDir), "", 0, fetcher),
		},
	}
	catalog := costruntime.NewProviderCatalog(router, registry, map[string]model.ProviderMetadata{
		awskit.ProviderID: {
			DisplayName: def.Manifest.DisplayName,
			PriceSource: def.Manifest.PriceSource,
		},
	})
	runtimeRegistry := costruntime.NewProviderRuntimeRegistry(runtimes)
	e := engine.NewEstimatorWithCatalogAndRuntimeRegistry(catalog, runtimeRegistry)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "unsupported_resource"))

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	rc := enginetest.AssertUnsupportedResource(t, result.Resources, "aws_cloudfront_distribution.main", awskit.ProviderID, model.CostErrorNoHandler)
	if rc.ErrorDetail != "no handler" {
		t.Errorf("ErrorDetail = %q, want %q", rc.ErrorDetail, "no handler")
	}
	if result.Unsupported != 1 {
		t.Errorf("Unsupported = %d, want 1", result.Unsupported)
	}
}

func TestEstimateModule_UsageBasedResource(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "usage_based"))

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	enginetest.AssertUsageBasedResource(t, result.Resources, "aws_sqs_queue.main")
	// Usage-based should NOT increment Unsupported counter
	if result.Unsupported != 0 {
		t.Errorf("Unsupported = %d, want 0 (usage-based is not unsupported)", result.Unsupported)
	}
}

func TestEstimateModule_FixedCostResource(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "fixed_cost"))

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	enginetest.AssertFixedResource(t, result.Resources, "aws_route53_zone.main", 0.50, 0.01)
	enginetest.AssertCostNear(t, "AfterCost", result.AfterCost, 0.50, 0.01)
}

func TestEstimateModule_MixedActions(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "mixed_actions"))

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	// aws_instance.new (create): ~$7.59 + ~$0.80 root vol = ~$8.39
	// aws_route53_zone.dns (no-op): $0.50 in before AND after
	// aws_sqs_queue.events (create, usage-based): $0
	// Expected: AfterCost ≈ $8.39 + $0.50 = $8.89, BeforeCost ≈ $0.50
	enginetest.AssertCostNear(t, "AfterCost", result.AfterCost, 8.892, 0.05)
	enginetest.AssertCostNear(t, "BeforeCost", result.BeforeCost, 0.50, 0.01)

	// Verify resource count: instance + root_vol + route53 + sqs = 4
	if len(result.Resources) != 4 {
		t.Errorf("resources count = %d, want 4", len(result.Resources))
	}
}

func TestEstimateModule_CompoundResource(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "create_ec2"))

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
			enginetest.AssertCostNear(t, "root_volume cost", rc.MonthlyCost, 0.80, 0.01) // 8GB * $0.10
			break
		}
	}
	if !found {
		t.Error("CompoundHandler did not synthesize root_block_device")
	}
}

func TestEstimateModule_SubmoduleGrouping(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "with_modules"))

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
			enginetest.AssertCostNear(t, "vpc cost", sm.MonthlyCost, 0.50, 0.01)
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
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "empty"))

	result, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err != nil {
		t.Fatalf("EstimateModule: %v", err)
	}

	if len(result.Resources) != 0 {
		t.Errorf("resources = %d, want 0", len(result.Resources))
	}
	enginetest.AssertCostNear(t, "AfterCost", result.AfterCost, 0, 0.001)
	enginetest.AssertCostNear(t, "BeforeCost", result.BeforeCost, 0, 0.001)
	if result.HasChanges {
		t.Error("HasChanges should be false for empty plan")
	}
}

func TestEstimateModule_InvalidPlanJSON(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, `{"invalid": "json"`)

	_, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err == nil {
		t.Fatal("expected error for invalid plan.json")
	}
}

func TestEstimateModule_NoPlanFile(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := t.TempDir()

	_, err := e.EstimateModule(context.Background(), dir, "us-east-1")
	if err == nil {
		t.Fatal("expected error when plan.json doesn't exist")
	}
}

func TestEstimateModules_Concurrent(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	tmpDir := t.TempDir()

	modules := []string{"vpc", "rds", "eks"}
	paths := make([]string, len(modules))
	regions := make(map[string]string)

	for i, mod := range modules {
		dir := filepath.Join(tmpDir, "platform", "prod", "us-east-1", mod)
		enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "create_ec2"))
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
	enginetest.AssertCostNear(t, "TotalAfter", result.TotalAfter, result.Modules[0].AfterCost*3, 0.01)
	if result.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", result.Currency)
	}
	if result.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
}

func TestEstimateModules_PartialFailure(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	tmpDir := t.TempDir()

	// One valid module, one with invalid plan
	validDir := filepath.Join(tmpDir, "valid")
	enginetest.WritePlan(t, validDir, enginetest.LoadPlanFixture(t, "create_ec2"))

	invalidDir := filepath.Join(tmpDir, "invalid")
	enginetest.WritePlan(t, invalidDir, `not json at all`)

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
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "create_ec2"))

	// Pass empty regions map — should use DefaultRegion
	result, err := e.EstimateModules(context.Background(), []string{dir}, map[string]string{})
	if err != nil {
		t.Fatalf("EstimateModules: %v", err)
	}

	if len(result.Modules) != 1 {
		t.Fatalf("modules = %d, want 1", len(result.Modules))
	}
	if result.Modules[0].Region != engine.DefaultRegion {
		t.Errorf("Region = %q, want %q", result.Modules[0].Region, engine.DefaultRegion)
	}
}

func TestValidateAndPrefetch_DownloadsMissing(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "create_ec2"))

	regions := map[string]string{dir: "us-east-1"}
	err := e.ValidateAndPrefetch(context.Background(), []string{dir}, regions)
	if err != nil {
		t.Fatalf("ValidateAndPrefetch: %v", err)
	}

	// After prefetch, cache should have EC2 pricing
	entries := e.CacheEntries(context.Background())
	if len(entries) == 0 {
		t.Error("cache should have entries after prefetch")
	}
}

func TestValidateAndPrefetch_SkipsUsageBased(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "usage_based"))

	regions := map[string]string{dir: "us-east-1"}
	err := e.ValidateAndPrefetch(context.Background(), []string{dir}, regions)
	if err != nil {
		t.Fatalf("ValidateAndPrefetch: %v", err)
	}

	// Usage-based resources should not trigger any downloads
	entries := e.CacheEntries(context.Background())
	if len(entries) != 0 {
		t.Errorf("cache entries = %d, want 0 (no standard resources to fetch)", len(entries))
	}
}

func TestValidateAndPrefetch_SkipsCachedData(t *testing.T) {
	e := enginetest.NewTestEstimator(t)
	dir := filepath.Join(t.TempDir(), "mod")
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "create_ec2"))

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
	e := engine.NewEstimator(diskblob.NewStore(t.TempDir()), "", 0, awskit.NewFetcher())
	if e == nil {
		t.Fatal("NewEstimator returned nil")
	}
}

func TestNewEstimatorFromConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		e, err := engine.NewEstimatorFromConfig(nil, diskblob.NewStore(t.TempDir()))
		if err != nil {
			t.Fatalf("NewEstimatorFromConfig() error = %v", err)
		}
		if e == nil {
			t.Fatal("expected non-nil estimator")
		}
	})

	t.Run("custom cache dir and TTL", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &model.CostConfig{
			BlobCache: &model.BlobCacheConfig{
				TTL: "2h",
			},
			Providers: model.CostProvidersConfig{
				AWS: &model.ProviderConfig{Enabled: true},
			},
		}
		e, err := engine.NewEstimatorFromConfig(cfg, diskblob.NewStore(tmpDir))
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
		cfg := &model.CostConfig{
			BlobCache: &model.BlobCacheConfig{
				TTL: "invalid",
			},
			Providers: model.CostProvidersConfig{
				AWS: &model.ProviderConfig{Enabled: true},
			},
		}
		e, err := engine.NewEstimatorFromConfig(cfg, diskblob.NewStore(t.TempDir()))
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
	e := engine.NewEstimator(diskblob.NewStore(cacheDir), "", 0, awskit.NewFetcher())

	if e.CacheDir() != cacheDir {
		t.Errorf("CacheDir() = %q, want %q", e.CacheDir(), cacheDir)
	}
	if e.CacheTTL() <= 0 {
		t.Errorf("CacheTTL() = %v, want > 0", e.CacheTTL())
	}
	if e.CacheOldestAge(context.Background()) != 0 {
		t.Errorf("CacheOldestAge() = %v, want 0 for empty cache", e.CacheOldestAge(context.Background()))
	}
	if len(e.CacheEntries(context.Background())) != 0 {
		t.Errorf("CacheEntries() len = %d, want 0", len(e.CacheEntries(context.Background())))
	}
	e.CleanExpiredCache(context.Background()) // should not panic
}

func TestAggregateCost(t *testing.T) {
	tests := []struct {
		name       string
		action     engine.EstimateAction
		cost       float64
		wantBefore float64
		wantAfter  float64
	}{
		{"create", engine.ActionCreate, 10, 0, 10},
		{"delete", engine.ActionDelete, 10, 10, 0},
		{"update", engine.ActionUpdate, 10, 10, 10},
		{"replace", engine.ActionReplace, 10, 10, 10},
		{"no-op", engine.ActionNoOp, 10, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &model.ModuleCost{}
			rc := model.ResourceCost{MonthlyCost: tt.cost, BeforeMonthlyCost: tt.cost}
			engine.AggregateCost(result, rc, tt.action)

			enginetest.AssertCostNear(t, "BeforeCost", result.BeforeCost, tt.wantBefore, 0.001)
			enginetest.AssertCostNear(t, "AfterCost", result.AfterCost, tt.wantAfter, 0.001)
		})
	}
}

func TestAggregateCost_UnsupportedNotCounted(t *testing.T) {
	result := &model.ModuleCost{}
	rc := model.ResourceCost{
		MonthlyCost: 100,
		ErrorKind:   model.CostErrorNoHandler,
	}
	engine.AggregateCost(result, rc, engine.ActionCreate)

	if result.AfterCost != 0 {
		t.Errorf("AfterCost = %.2f, want 0 (unsupported should not add cost)", result.AfterCost)
	}
	if result.Unsupported != 1 {
		t.Errorf("Unsupported = %d, want 1", result.Unsupported)
	}
}
