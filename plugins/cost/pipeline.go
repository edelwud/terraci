package cost

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
)

// PipelineContribution adds a cost estimation job to the CI pipeline.
func (p *Plugin) PipelineContribution() *pipeline.Contribution {
	if !p.IsConfigured() {
		return nil
	}

	serviceDir := p.serviceDirRel
	return &pipeline.Contribution{
		Jobs: []pipeline.ContributedJob{{
			Name:          "cost-estimation",
			Phase:         pipeline.PhasePostPlan,
			Commands:      []string{"terraci cost"},
			ArtifactPaths: []string{filepath.Join(serviceDir, "cost-results.json")},
			DependsOnPlan: true,
			AllowFailure:  true,
		}},
	}
}
