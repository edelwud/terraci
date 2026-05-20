package policy

import (
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

const (
	jobName = "policy-check"
)

var (
	resultsFile = ci.ResultFilename(pluginName)
	reportFile  = ci.ReportFilename(pluginName)
)

// PipelineContribution adds a policy-check job to the pipeline DAG.
// Framework guarantees this is only called when IsEnabled() == true.
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution {
	serviceDir := ""
	if ctx != nil && ctx.Config().Present() {
		serviceDir = ctx.Config().ServiceDir()
	}
	allowFailure := true
	if cfg := p.Config(); cfg != nil {
		allowFailure = !cfg.CanBlock()
	}
	job, err := pipeline.NewPluginCommandJob(pipeline.PluginCommandJobOptions{
		Name:     jobName,
		Commands: []string{"terraci policy check --format text"},
		Consumes: []pipeline.ResourceRequest{
			pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
		},
		Produces:     pipeline.PluginResultAndReportResources(serviceDir, pluginName),
		AllowFailure: allowFailure,
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
