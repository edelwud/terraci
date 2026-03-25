package policy

import (
	"github.com/edelwud/terraci/pkg/pipeline"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

// PipelineContribution adds a policy-check job to the CI pipeline.
func (p *Plugin) PipelineContribution() *pipeline.Contribution {
	if !p.IsConfigured() || p.cfg == nil || !p.cfg.Enabled {
		return nil
	}
	allowFailure := p.cfg.OnFailure == policyengine.ActionWarn
	return &pipeline.Contribution{
		Jobs: []pipeline.ContributedJob{{
			Name:          "policy-check",
			Phase:         pipeline.PhasePostPlan,
			Commands:      []string{"terraci policy pull", "terraci policy check"},
			ArtifactPaths: []string{".terraci/policy-results.json"},
			DependsOnPlan: true,
			AllowFailure:  allowFailure,
		}},
	}
}
