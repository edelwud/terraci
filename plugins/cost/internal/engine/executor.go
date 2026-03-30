package engine

import (
	"context"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

// ModuleExecutor executes scanned module plans through the cost resolver.
type ModuleExecutor struct {
	resolver *costruntime.CostResolver
}

// NewModuleExecutor creates a module executor for the provided resolver.
func NewModuleExecutor(resolver *costruntime.CostResolver) *ModuleExecutor {
	return &ModuleExecutor{resolver: resolver}
}

// Execute resolves all resources in a module plan and aggregates the result.
func (e *ModuleExecutor) Execute(ctx context.Context, modulePlan *ModulePlan) *model.ModuleCost {
	result := &model.ModuleCost{
		ModuleID:   modulePlan.ModuleID,
		ModulePath: modulePlan.ModulePath,
		Region:     modulePlan.Region,
		Resources:  make([]model.ResourceCost, 0),
		HasChanges: modulePlan.HasChanges,
	}

	for _, resource := range modulePlan.Resources {
		req := resource.ResolveRequest(modulePlan.Region)
		costs := e.resolver.ResolveWithSubResources(ctx, req)

		for i := range costs {
			if i == 0 && resource.RequiresBeforeCost() {
				e.resolver.ResolveBeforeCost(ctx, &costs[i], resource.ResourceType, resource.BeforeAttrs, modulePlan.Region)
			}

			result.Resources = append(result.Resources, costs[i])
			aggregateCost(result, costs[i], resource.Action)
		}
	}

	result.DiffCost = result.AfterCost - result.BeforeCost
	result.Submodules = model.GroupByModule(result.Resources)
	result.Provider, result.Providers = summarizeProviders(result.Resources)

	return result
}

func aggregateCost(result *model.ModuleCost, rc model.ResourceCost, action EstimateAction) {
	if rc.IsUnsupported() {
		result.Unsupported++
		return
	}

	switch action {
	case ActionCreate:
		result.AfterCost += rc.MonthlyCost
	case ActionDelete:
		result.BeforeCost += rc.MonthlyCost
	case ActionUpdate, ActionReplace:
		result.BeforeCost += rc.BeforeMonthlyCost
		result.AfterCost += rc.MonthlyCost
	case ActionNoOp:
		result.BeforeCost += rc.MonthlyCost
		result.AfterCost += rc.MonthlyCost
	}
}

// AggregateCost exposes module cost aggregation for tests and facades.
func AggregateCost(result *model.ModuleCost, rc model.ResourceCost, action EstimateAction) {
	aggregateCost(result, rc, action)
}

func summarizeProviders(resources []model.ResourceCost) (primary string, providers []string) {
	providerSet := make(map[string]bool)
	for i := range resources {
		resource := &resources[i]
		if resource.Provider != "" {
			providerSet[resource.Provider] = true
		}
	}

	if len(providerSet) == 0 {
		return "", nil
	}

	providers = make([]string, 0, len(providerSet))
	for providerID := range providerSet {
		providers = append(providers, providerID)
	}
	if len(providers) == 1 {
		return providers[0], providers
	}
	return "", providers
}
