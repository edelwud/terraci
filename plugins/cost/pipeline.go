package cost

import (
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

const (
	jobName = "cost-estimation"
)

var (
	resultsFile = ci.ResultFilename(pluginName)
	reportFile  = ci.ReportFilename(pluginName)
)

// PipelineContribution adds a cost estimation job to the pipeline DAG.
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
			Commands: []string{"terraci cost"},
			Consumes: []pipeline.ResourceRequest{
				pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
			},
			Produces: pipeline.PluginResultAndReportResources(serviceDir, pluginName),
			// AllowFailure lets the pipeline proceed even when cost estimation fails
			// (e.g., missing AWS credentials or unsupported resource types).
			AllowFailure: true,
		}},
	}
}
