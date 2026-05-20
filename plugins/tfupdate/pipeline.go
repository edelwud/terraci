package tfupdate

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

func (p *Plugin) PipelineContributionEnabled(_ *plugin.AppContext) (bool, error) {
	cfg := p.Config()
	return cfg != nil && cfg.Pipeline, nil
}

// PipelineContribution adds a dependency update check job to the pipeline DAG.
// Only contributes when pipeline: true is set in config.
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) (*pipeline.Contribution, error) {
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
		return nil, fmt.Errorf("build tfupdate pipeline job: %w", err)
	}
	contribution, err := pipeline.NewContribution(job)
	if err != nil {
		return nil, fmt.Errorf("build tfupdate pipeline contribution: %w", err)
	}
	return contribution, nil
}
