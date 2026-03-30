package cost

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

const (
	resultsFile = "cost-results.json"
	reportFile  = "cost-report.json"
)

// PipelineContribution adds a cost estimation job to the CI pipeline.
// Framework guarantees this is only called when IsEnabled() == true.
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution {
	serviceDir := ""
	if cfg := ctx.Config(); cfg != nil {
		serviceDir = cfg.ServiceDir
	}
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
