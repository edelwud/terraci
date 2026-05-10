package policy

import (
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

const (
	jobName = "policy-check"
)

var (
	resultsFile = ci.ResultFilename(pluginName)
	reportFile  = ci.ReportFilename(pluginName)
)

// PipelineContribution adds a policy-check job to the pipeline DAG.
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
			Commands: []string{"terraci policy pull", "terraci policy check"},
			Consumes: []pipeline.ResourceRequest{
				pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
			},
			Produces:     pipeline.PluginResultAndReportResources(serviceDir, pluginName),
			AllowFailure: allowFailure,
		}},
	}
}
