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
	if ctx == nil || !ctx.Config().Present() {
		return nil
	}
	serviceDir := ctx.Config().ServiceDir()
	job, err := pipeline.NewPluginCommandJob(pipeline.PluginCommandJobOptions{
		Name:     jobName,
		Commands: []string{"terraci cost"},
		Consumes: []pipeline.ResourceRequest{
			pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
		},
		Produces: pipeline.PluginResultAndReportResources(serviceDir, pluginName),
		// AllowFailure lets the pipeline proceed even when cost estimation fails
		// (e.g., missing AWS credentials or unsupported resource types).
		AllowFailure: true,
	})
	if err != nil {
		return nil
	}
	contribution, err := pipeline.NewContribution(job)
	if err != nil {
		return nil
	}
	return contribution
}
