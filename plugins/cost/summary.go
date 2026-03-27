package cost

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	costengine "github.com/edelwud/terraci/plugins/cost/internal"
)

// ContributeToSummary loads cost estimation results from the service directory
// and enriches plan results with cost data.
func (p *Plugin) ContributeToSummary(_ context.Context, appCtx *plugin.AppContext, execCtx *plugin.ExecutionContext) error {
	if !p.IsConfigured() || !p.cfg.Enabled {
		return nil
	}

	path := filepath.Join(appCtx.ServiceDir, "cost-results.json")
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		log.WithField("path", path).Debug("no cost results found, skipping")
		return nil //nolint:nilerr // missing file is expected — cost job may not have run
	}

	var result costengine.EstimateResult
	if err := json.Unmarshal(data, &result); err != nil {
		log.WithError(err).Warn("failed to parse cost results")
		return nil
	}

	collection := execCtx.PlanResults
	if collection == nil || len(collection.Results) == 0 {
		return nil
	}

	// Enrich plan results with cost data
	costByModule := make(map[string]int, len(result.Modules))
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
	execCtx.SetData("cost:result", &result)

	log.WithField("modules", len(result.Modules)).Info("loaded cost estimation results")
	return nil
}
