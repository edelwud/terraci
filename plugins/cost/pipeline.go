package cost

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

const (
	jobName     = "cost-estimation"
	resultsFile = "cost-results.json"
	reportFile  = "cost-report.json"
)

// PipelineContribution adds a cost estimation job to the CI pipeline.
// Framework guarantees this is only called when IsEnabled() == true.
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution {
	cfg := ctx.Config()
	if cfg == nil {
		return nil
	}
	serviceDir := cfg.ServiceDir
	return &pipeline.Contribution{
		// `terraci cost` reads plan.json from each module directory, so the
		// generator must enable detailed plan output regardless of MR/PR
		// comment configuration.
		RequiresDetailedPlan: true,
		Jobs: []pipeline.ContributedJob{{
			Name:          jobName,
			Phase:         pipeline.PhasePostPlan,
			Commands:      []string{"terraci cost"},
			Artifact:      pipeline.ResultArtifact(jobName, filepath.Join(serviceDir, resultsFile), filepath.Join(serviceDir, reportFile)),
			DependsOnPlan: true,
			// AllowFailure lets the pipeline proceed even when cost estimation fails
			// (e.g., missing AWS credentials or unsupported resource types).
			AllowFailure: true,
		}},
	}
}
