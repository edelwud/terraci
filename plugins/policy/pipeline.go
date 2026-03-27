package policy

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

// PipelineContribution adds a policy-check job to the CI pipeline.
func (p *Plugin) PipelineContribution() *pipeline.Contribution {
	if !p.IsConfigured() {
		return nil
	}
	serviceDir := p.serviceDirRel
	allowFailure := p.cfg.OnFailure == policyengine.ActionWarn
	return &pipeline.Contribution{
		Jobs: []pipeline.ContributedJob{{
			Name:          "policy-check",
			Phase:         pipeline.PhasePostPlan,
			Commands:      []string{"terraci policy pull", "terraci policy check"},
			ArtifactPaths: []string{filepath.Join(serviceDir, "policy-results.json")},
			DependsOnPlan: true,
			AllowFailure:  allowFailure,
		}},
	}
}
