package cost

import (
	"errors"
	"fmt"

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
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) (*pipeline.Contribution, error) {
	if ctx == nil || !ctx.Config().Present() {
		return nil, errors.New("app config is required")
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
		return nil, fmt.Errorf("build cost pipeline job: %w", err)
	}
	contribution, err := pipeline.NewContribution(job)
	if err != nil {
		return nil, fmt.Errorf("build cost pipeline contribution: %w", err)
	}
	return contribution, nil
}
