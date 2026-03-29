package policy

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

const resultsFile = "policy-results.json"

// PipelineContribution adds a policy-check job to the CI pipeline.
// Framework guarantees this is only called when IsEnabled() == true.
func (p *Plugin) PipelineContribution() *pipeline.Contribution {
	serviceDir := p.serviceDirRel
	allowFailure := p.Config().OnFailure == policyengine.ActionWarn
	return &pipeline.Contribution{
		Jobs: []pipeline.ContributedJob{{
			Name:          "policy-check",
			Phase:         pipeline.PhasePostPlan,
			Commands:      []string{"terraci policy pull", "terraci policy check"},
			ArtifactPaths: []string{filepath.Join(serviceDir, resultsFile)},
			DependsOnPlan: true,
			AllowFailure:  allowFailure,
		}},
	}
}
