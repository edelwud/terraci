package summary

import (
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

// PipelineContribution returns the summary job contribution.
// Framework guarantees this is only called when IsEnabled() == true.
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
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
