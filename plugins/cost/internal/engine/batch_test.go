package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

type stubCatalog struct {
	providers map[resourcedef.ResourceType]string
	defs      map[resourcedef.ResourceType]resourcedef.Definition
}

func (c stubCatalog) ResolveProvider(resourceType resourcedef.ResourceType) (string, bool) {
	providerID, ok := c.providers[resourceType]
	return providerID, ok
}

func (c stubCatalog) ResolveDefinition(_ string, resourceType resourcedef.ResourceType) (resourcedef.Definition, bool) {
	def, ok := c.defs[resourceType]
	return def, ok
}

type stubStandardDefinition struct {
	lookup *pricing.PriceLookup
	err    error
}

type stubAdapter struct {
	results map[string]stubAdapterResult
}

type stubAdapterResult struct {
	plan *ModulePlan
	err  error
}

func (h stubStandardDefinition) Definition(resourceType resourcedef.ResourceType) resourcedef.Definition {
	return resourcedef.Definition{
		Type:     resourceType,
		Category: resourcedef.CostCategoryStandard,
		Lookup: func(string, map[string]any) (*pricing.PriceLookup, error) {
			return h.lookup, h.err
		},
		StandardCost: func(*pricing.Price, *pricing.PriceIndex, string, map[string]any) (hourly, monthly float64) {
			return 0, 0
		},
	}
}

func (a stubAdapter) LoadModule(modulePath, _ string) (*ModulePlan, error) {
	result, ok := a.results[modulePath]
	if !ok {
		return nil, fmt.Errorf("unexpected module %s", modulePath)
	}
	return result.plan, result.err
}

func TestBuildPrefetchPlan_CollectsDiagnosticsAndRequirements(t *testing.T) {
	t.Parallel()

	plan := &ModulePlan{
		ModuleID: "svc/prod/us-east-1/app",
		Region:   "us-east-1",
		Resources: []PlannedResource{
			{ResourceType: "aws_instance", Address: "aws_instance.ok"},
			{ResourceType: "aws_db_instance", Address: "aws_db_instance.bad_lookup"},
			{ResourceType: "aws_cloudfront_distribution", Address: "aws_cloudfront_distribution.unknown"},
			{ResourceType: "aws_secretsmanager_secret", Address: "aws_secretsmanager_secret.no_handler"},
		},
	}

	catalog := stubCatalog{
		providers: map[resourcedef.ResourceType]string{
			"aws_instance":              "aws",
			"aws_db_instance":           "aws",
			"aws_secretsmanager_secret": "aws",
		},
		defs: map[resourcedef.ResourceType]resourcedef.Definition{
			"aws_instance": stubStandardDefinition{
				lookup: &pricing.PriceLookup{ServiceID: pricing.ServiceID{Provider: "aws", Name: "AmazonEC2"}},
			}.Definition("aws_instance"),
			"aws_db_instance": stubStandardDefinition{
				err: fmt.Errorf("missing instance_class"),
			}.Definition("aws_db_instance"),
		},
	}

	runtime := newStubEstimationRuntime(t, catalog)
	prefetch := buildPrefetchPlan(runtime, []*ModulePlan{plan})

	if got := prefetch.services[pricing.ServiceID{Provider: "aws", Name: "AmazonEC2"}]; len(got) != 1 || got[0] != "us-east-1" {
		t.Fatalf("prefetch.services = %#v, want EC2 us-east-1 warm requirement", prefetch.services)
	}
	if len(prefetch.diagnostics) != 3 {
		t.Fatalf("len(prefetch.diagnostics) = %d, want 3", len(prefetch.diagnostics))
	}
}

func TestEstimate_AssignsPrefetchWarningsToResult(t *testing.T) {
	t.Parallel()

	coord := &estimateCoordinator{
		scanner: NewModuleScanner(stubAdapter{
			results: map[string]stubAdapterResult{
				"mod-a": {
					plan: &ModulePlan{
						ModuleID: "mod-a",
						Region:   "us-east-1",
						Resources: []PlannedResource{
							{ResourceType: "aws_cloudfront_distribution", Address: "aws_cloudfront_distribution.main"},
						},
					},
				},
			},
		}),
		executor: NewModuleExecutor(noopResolver{}),
		runtime:  newStubEstimationRuntime(t, stubCatalog{}),
	}

	result, err := coord.Estimate(context.Background(), []string{"mod-a"}, map[string]string{"mod-a": "us-east-1"})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	if len(result.PrefetchWarnings) != 1 {
		t.Fatalf("len(result.PrefetchWarnings) = %d, want 1", len(result.PrefetchWarnings))
	}
}

type noopResolver struct{}

func (noopResolver) ResolveWithSubResourcesState(context.Context, costruntime.ResolveRequest, *costruntime.ResolutionState) []model.ResourceCost {
	return nil
}

func (noopResolver) ResolveBeforeCostWithState(context.Context, *model.ResourceCost, resourcedef.ResourceType, map[string]any, string, *costruntime.ResolutionState) {
}

func newStubEstimationRuntime(t testing.TB, stub stubCatalog) *costruntime.EstimationRuntime {
	t.Helper()

	router := costruntime.NewResourceProviderRouter()
	defs := make(map[string]map[resourcedef.ResourceType]resourcedef.Definition)
	for resourceType, providerID := range stub.providers {
		router.Register(providerID, resourceType)
	}
	for resourceType, def := range stub.defs {
		providerID, ok := stub.providers[resourceType]
		if !ok {
			continue
		}
		if defs[providerID] == nil {
			defs[providerID] = make(map[resourcedef.ResourceType]resourcedef.Definition)
		}
		defs[providerID][resourceType] = def
	}

	runtime, err := costruntime.NewEstimationRuntime(
		costruntime.NewProviderCatalog(router, defs, nil),
		costruntime.NewProviderRuntimeRegistry(nil),
	)
	if err != nil {
		t.Fatalf("NewEstimationRuntime() error = %v", err)
	}
	return runtime
}
