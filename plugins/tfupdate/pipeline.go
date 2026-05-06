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
			Name:          jobName,
			Phase:         pipeline.PhasePrePlan,
			Commands:      []string{"terraci tfupdate"},
			Artifact:      pipeline.ResultArtifact(jobName, filepath.Join(serviceDir, resultsFile), filepath.Join(serviceDir, reportFile)),
			DependsOnPlan: false,
			AllowFailure:  true,
		}},
	}
}
