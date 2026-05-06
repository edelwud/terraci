package cost

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

const (
	jobName     = "cost-estimation"
	resultsFile = "cost-results.json"
	reportFile  = "cost-report.json"
)

// PipelineContribution adds a cost estimation job to the CI pipeline.
// Framework guarantees this is only called when IsEnabled() == true.
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution {
	cfg := ctx.Config()
	if cfg == nil {
		return nil
	}
	serviceDir := cfg.ServiceDir
	return &pipeline.Contribution{
		Jobs: []pipeline.ContributedJob{{
			Name:     jobName,
			Phase:    pipeline.PhasePostPlan,
			Commands: []string{"terraci cost"},
			Consumes: []pipeline.ResourceRequest{
				pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
			},
			Produces: []pipeline.ResourceSpec{
				pipeline.PluginResource(pipeline.ResourceKindPluginResult, pluginName, filepath.Join(serviceDir, resultsFile)),
				pipeline.PluginResource(pipeline.ResourceKindPluginReport, pluginName, filepath.Join(serviceDir, reportFile)),
			},
			// AllowFailure lets the pipeline proceed even when cost estimation fails
			// (e.g., missing AWS credentials or unsupported resource types).
			AllowFailure: true,
		}},
	}
}
