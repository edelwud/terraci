package tfupdate

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

// PipelineContribution adds a dependency update check job to the CI pipeline.
// Only contributes when pipeline: true is set in config.
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution {
	if cfg := p.Config(); cfg == nil || !cfg.Pipeline {
		return nil
	}

	const jobName = "tfupdate-check"
	serviceDir := ""
	if cfg := ctx.Config(); cfg != nil {
		serviceDir = cfg.ServiceDir
	}
	return &pipeline.Contribution{
		Jobs: []pipeline.ContributedJob{{
			Name:     jobName,
			Phase:    pipeline.PhasePrePlan,
			Commands: []string{"terraci tfupdate"},
			Produces: []pipeline.ResourceSpec{
				pipeline.PluginResource(pipeline.ResourceKindPluginResult, pluginName, filepath.Join(serviceDir, resultsFile)),
				pipeline.PluginResource(pipeline.ResourceKindPluginReport, pluginName, filepath.Join(serviceDir, reportFile)),
			},
			AllowFailure: true,
		}},
	}
}
