package update

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
)

// PipelineContribution adds a dependency update check job to the CI pipeline.
// Only contributes when pipeline: true is set in config.
func (p *Plugin) PipelineContribution() *pipeline.Contribution {
	if cfg := p.Config(); cfg == nil || !cfg.Pipeline {
		return nil
	}

	serviceDir := p.serviceDirRel
	return &pipeline.Contribution{
		Jobs: []pipeline.ContributedJob{{
			Name:          "dependency-update-check",
			Phase:         pipeline.PhasePrePlan,
			Commands:      []string{"terraci update"},
			ArtifactPaths: []string{filepath.Join(serviceDir, resultsFile)},
			DependsOnPlan: false,
			AllowFailure:  true,
		}},
	}
}
