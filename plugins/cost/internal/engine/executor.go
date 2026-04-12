package engine

import (
	"context"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/results"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

// moduleResolver is the narrow interface that ModuleExecutor requires from the cost resolver.
type moduleResolver interface {
	ResolveWithSubResourcesState(ctx context.Context, req costruntime.ResolveRequest, state *costruntime.ResolutionState) []model.ResourceCost
	ResolveBeforeCostWithState(ctx context.Context, rc *model.ResourceCost, resourceType resourcedef.ResourceType, beforeAttrs map[string]any, region string, state *costruntime.ResolutionState)
}

// ModuleExecutor executes scanned module plans through the cost resolver.
type ModuleExecutor struct {
	resolver moduleResolver
}

// NewModuleExecutor creates a module executor for the provided resolver.
func NewModuleExecutor(resolver moduleResolver) *ModuleExecutor {
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
		state := costruntime.NewResolutionState()
		req := resource.ResolveRequest(modulePlan.Region)
		costs := e.resolver.ResolveWithSubResourcesState(ctx, req, state)

		for i := range costs {
			if i == 0 && resource.RequiresBeforeCost() {
				e.resolver.ResolveBeforeCostWithState(ctx, &costs[i], resource.ResourceType, resource.BeforeAttrs, modulePlan.Region, state)
			}
			assembler.AddResource(costs[i], resource.Action)
		}
	}

	return assembler.Build()
}
