package generate

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/pipelinetest"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

// BuildPipelineIR builds the canonical pipeline IR for the GitLab generator
// from raw inputs. Provider-specific knobs (e.g. DetailedPlan inferred from
// "MR comments enabled?") are derived from the GitLab settings; the actual
// IR construction goes through pipelinetest.BuildIR so the gitlab and
// github helpers stay in lockstep.
func BuildPipelineIR(
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
) (*pipeline.IR, error) {
	s := newSettings(cfg, execCfg)
	requiredResources := providerCommentResources(s.mrCommentEnabled())
	return pipelinetest.BuildIR(pipelinetest.IROptions{
		Script: pipeline.ScriptConfig{
			InitEnabled: s.initEnabled(),
			PlanEnabled: s.planEnabled(),
			AutoApprove: s.autoApprove(),
			// execution.plan_mode=detailed is an explicit user/runtime request.
			// MR comments declare plan.txt + plan.json via RequiredResources.
			DetailedPlan: execCfg.PlanMode == execution.PlanModeDetailed,
		},
		Contributions:     contributions,
		RequiredResources: requiredResources,
		DepGraph:          depGraph,
		AllModules:        allModules,
		TargetModules:     targetModules,
		PlanEnabled:       s.planEnabled(),
		PlanOnly:          s.planOnly(),
	})
}

func providerCommentResources(enabled bool) []pipeline.ResourceRequest {
	if !enabled {
		return nil
	}
	return []pipeline.ResourceRequest{
		pipeline.AllPlanResources(pipeline.ResourceKindPlanText),
		pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
	}
}
