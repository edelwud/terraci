package summary

import (
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

// PipelineContribution returns the summary job contribution.
// Framework guarantees this is only called when IsEnabled() == true.
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
	return &pipeline.Contribution{
		// `terraci summary` ingests plan.json from each module via
		// planresults.Scan, so detailed plan output is mandatory whenever
		// summary participates in the pipeline.
		RequiresDetailedPlan: true,
		Jobs: []pipeline.ContributedJob{{
			Name:          "terraci-summary",
			Phase:         pipeline.PhaseFinalize,
			Commands:      []string{"terraci summary"},
			DependsOnPlan: true,
			AllowFailure:  false,
		}},
	}
}
