package cost

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
)

// ContributeToSummary enriches plan results with cost data during summary.
func (p *Plugin) ContributeToSummary(ctx context.Context, _ *plugin.AppContext, execCtx *plugin.ExecutionContext) error {
	if !p.IsConfigured() || !p.cfg.Enabled {
		return nil
	}

	collection := execCtx.PlanResults
	if collection == nil || len(collection.Results) == 0 {
		return nil
	}

	// Build module paths and regions
	modulePaths := make([]string, 0, len(collection.Results))
	regions := make(map[string]string)
	for i := range collection.Results {
		r := &collection.Results[i]
		modulePaths = append(modulePaths, r.ModulePath)
		if region := r.Get("region"); region != "" {
			regions[r.ModulePath] = region
		}
	}

	est := p.getEstimator()

	// Prefetch pricing
	if err := est.ValidateAndPrefetch(ctx, modulePaths, regions); err != nil {
		return fmt.Errorf("prefetch pricing: %w", err)
	}

	// Estimate costs
	result, err := est.EstimateModules(ctx, modulePaths, regions)
	if err != nil {
		return fmt.Errorf("estimate costs: %w", err)
	}

	// Enrich plan results with cost data
	costByModule := make(map[string]int)
	for i := range result.Modules {
		costByModule[result.Modules[i].ModulePath] = i
	}
	for i := range collection.Results {
		r := &collection.Results[i]
		if idx, ok := costByModule[r.ModulePath]; ok && result.Modules[idx].Error == "" {
			mc := &result.Modules[idx]
			r.CostBefore = mc.BeforeCost
			r.CostAfter = mc.AfterCost
			r.CostDiff = mc.DiffCost
			r.HasCost = true
		}
	}

	// Store result for other plugins
	execCtx.SetData("cost:result", result)

	return nil
}
