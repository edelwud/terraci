package summary

import "github.com/edelwud/terraci/pkg/pipeline"

// PipelineContribution returns the summary job contribution.
func (p *Plugin) PipelineContribution() *pipeline.Contribution {
	if !p.isEnabled() {
		return nil
	}
	return &pipeline.Contribution{
		Jobs: []pipeline.ContributedJob{{
			Name:          "terraci-summary",
			Phase:         pipeline.PhaseFinalize,
			Commands:      []string{"terraci summary"},
			DependsOnPlan: true,
			AllowFailure:  false,
		}},
	}
}
