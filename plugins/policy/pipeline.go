package policy

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

const (
	resultsFile = "policy-results.json"
	reportFile  = "policy-report.json"
)

// PipelineContribution adds a policy-check job to the CI pipeline.
// Framework guarantees this is only called when IsEnabled() == true.
func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution {
	serviceDir := ""
	if cfg := ctx.Config(); cfg != nil {
		serviceDir = cfg.ServiceDir
	}
	allowFailure := p.Config().OnFailure == policyengine.ActionWarn
	return &pipeline.Contribution{
		// `terraci policy check` reads plan.json from each module directory,
		// so detailed plan output must be on regardless of MR/PR comment
		// configuration.
		RequiresDetailedPlan: true,
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
