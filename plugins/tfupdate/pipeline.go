package tfupdate

import (
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

func (p *Plugin) PipelineContributionEnabled(_ *plugin.AppContext) bool {
	cfg := p.Config()
	return cfg != nil && cfg.Pipeline
}

// PipelineContribution adds a dependency update check job to the CI pipeline.
// Only contributes when pipeline: true is set in config.
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution {
	const jobName = "tfupdate-check"
	serviceDir := ""
	if cfg := ctx.Config(); cfg != nil {
		serviceDir = cfg.ServiceDir
	}
	return &pipeline.Contribution{
		Jobs: []pipeline.ContributedJob{{
			Name:         jobName,
			Commands:     []string{"terraci tfupdate"},
			Produces:     pipeline.PluginResultAndReportResources(serviceDir, pluginName),
			AllowFailure: true,
		}},
	}
}
