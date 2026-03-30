package engine

import (
	"context"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/results"
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
	assembler := results.NewModuleAssembler(results.ModuleIdentity{
		ModuleID:   modulePlan.ModuleID,
		ModulePath: modulePlan.ModulePath,
		Region:     modulePlan.Region,
		HasChanges: modulePlan.HasChanges,
	})

	for _, resource := range modulePlan.Resources {
		req := resource.ResolveRequest(modulePlan.Region)
		costs := e.resolver.ResolveWithSubResources(ctx, req)

		for i := range costs {
			if i == 0 && resource.RequiresBeforeCost() {
				e.resolver.ResolveBeforeCost(ctx, &costs[i], resource.ResourceType, resource.BeforeAttrs, modulePlan.Region)
			}
			assembler.AddResource(costs[i], resource.Action)
		}
	}

	return assembler.Build()
}

// AggregateCost exposes module cost aggregation for tests and facades.
func AggregateCost(result *model.ModuleCost, rc model.ResourceCost, action EstimateAction) {
	results.AggregateCost(result, rc, action)
}
