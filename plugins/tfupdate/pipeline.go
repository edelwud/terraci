package tfupdate

import (
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

func (p *Plugin) PipelineContributionEnabled(_ *plugin.AppContext) bool {
	cfg := p.Config()
	return cfg != nil && cfg.Pipeline
}

// PipelineContribution adds a dependency update check job to the pipeline DAG.
// Only contributes when pipeline: true is set in config.
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution {
	const jobName = "tfupdate-check"
	serviceDir := ""
	if ctx != nil && ctx.Config().Present() {
		serviceDir = ctx.Config().ServiceDir()
	}
	job, err := pipeline.NewPluginCommandJob(pipeline.PluginCommandJobOptions{
		Name:         jobName,
		Commands:     []string{"terraci tfupdate"},
		Produces:     pipeline.PluginResultAndReportResources(serviceDir, pluginName),
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
