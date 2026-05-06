package policy

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

const (
	pluginName  = "policy"
	jobName     = "policy-check"
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
		Jobs: []pipeline.ContributedJob{{
			Name:     jobName,
			Phase:    pipeline.PhasePostPlan,
			Commands: []string{"terraci policy pull", "terraci policy check"},
			Consumes: []pipeline.ResourceRequest{
				pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
			},
			Produces: []pipeline.ResourceSpec{
				pipeline.PluginResource(pipeline.ResourceKindPluginResult, pluginName, filepath.Join(serviceDir, resultsFile)),
				pipeline.PluginResource(pipeline.ResourceKindPluginReport, pluginName, filepath.Join(serviceDir, reportFile)),
			},
			AllowFailure: allowFailure,
		}},
	}
}
