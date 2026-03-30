package costengine

import (
	"context"

	"github.com/edelwud/terraci/internal/terraform/plan"
)

// ModuleExecutor executes scanned module plans through the cost resolver.
type ModuleExecutor struct {
	resolver *CostResolver
}

// NewModuleExecutor creates a module executor for the provided resolver.
func NewModuleExecutor(resolver *CostResolver) *ModuleExecutor {
	return &ModuleExecutor{resolver: resolver}
}

// Execute resolves all resources in a module plan and aggregates the result.
func (e *ModuleExecutor) Execute(ctx context.Context, modulePlan *ModulePlan) *ModuleCost {
	result := &ModuleCost{
		ModuleID:   modulePlan.ModuleID,
		ModulePath: modulePlan.ModulePath,
		Region:     modulePlan.Region,
		Resources:  make([]ResourceCost, 0),
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
	result.Submodules = groupByModule(result.Resources)
	result.Provider, result.Providers = summarizeProviders(result.Resources)

	return result
}

// aggregateCost adds a resource's cost to the module totals based on action.
func aggregateCost(result *ModuleCost, rc ResourceCost, action string) {
	if rc.IsUnsupported() {
		result.Unsupported++
		return
	}

	switch action {
	case plan.ActionCreate:
		result.AfterCost += rc.MonthlyCost
	case plan.ActionDelete:
		result.BeforeCost += rc.MonthlyCost
	case plan.ActionUpdate, plan.ActionReplace:
		result.BeforeCost += rc.BeforeMonthlyCost
		result.AfterCost += rc.MonthlyCost
	case plan.ActionNoOp:
		result.BeforeCost += rc.MonthlyCost
		result.AfterCost += rc.MonthlyCost
	}
}
