package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

type stubCatalog struct {
	providers map[handler.ResourceType]string
	handlers  map[handler.ResourceType]handler.ResourceHandler
}

func (c stubCatalog) ResolveProvider(resourceType handler.ResourceType) (string, bool) {
	providerID, ok := c.providers[resourceType]
	return providerID, ok
}

func (c stubCatalog) ResolveHandler(_ string, resourceType handler.ResourceType) (handler.ResourceHandler, bool) {
	h, ok := c.handlers[resourceType]
	return h, ok
}

type stubStandardHandler struct {
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

func (h stubStandardHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }
func (h stubStandardHandler) BuildLookup(string, map[string]any) (*pricing.PriceLookup, error) {
	return h.lookup, h.err
}
func (h stubStandardHandler) CalculateCost(*pricing.Price, *pricing.PriceIndex, string, map[string]any) (hourly, monthly float64) {
	return 0, 0
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
		providers: map[handler.ResourceType]string{
			"aws_instance":              "aws",
			"aws_db_instance":           "aws",
			"aws_secretsmanager_secret": "aws",
		},
		handlers: map[handler.ResourceType]handler.ResourceHandler{
			"aws_instance": stubStandardHandler{
				lookup: &pricing.PriceLookup{ServiceID: pricing.ServiceID{Provider: "aws", Name: "AmazonEC2"}},
			},
			"aws_db_instance": stubStandardHandler{
				err: fmt.Errorf("missing instance_class"),
			},
		},
	}

	prefetch := buildPrefetchPlan(catalog, []*ModulePlan{plan})

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
		executor:         NewModuleExecutor(noopResolver{}),
		providerMetadata: func() map[string]model.ProviderMetadata { return nil },
		catalog:          stubCatalog{},
		runtimes:         stubWarmer{},
	}

	result, err := coord.Estimate(context.Background(), []string{"mod-a"}, map[string]string{"mod-a": "us-east-1"})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	if len(result.PrefetchWarnings) != 1 {
		t.Fatalf("len(result.PrefetchWarnings) = %d, want 1", len(result.PrefetchWarnings))
	}
}

type stubWarmer struct{}

func (stubWarmer) WarmIndexes(context.Context, map[pricing.ServiceID][]string) error { return nil }

type noopResolver struct{}

func (noopResolver) ResolveWithSubResourcesState(context.Context, costruntime.ResolveRequest, *costruntime.ResolutionState) []model.ResourceCost {
	return nil
}

func (noopResolver) ResolveBeforeCostWithState(context.Context, *model.ResourceCost, handler.ResourceType, map[string]any, string, *costruntime.ResolutionState) {
}
