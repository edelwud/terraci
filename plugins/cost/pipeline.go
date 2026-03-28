package cost

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
)

const resultsFile = "cost-results.json"

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
			ArtifactPaths: []string{filepath.Join(serviceDir, resultsFile)},
			DependsOnPlan: true,
			AllowFailure:  true,
		}},
	}
}
