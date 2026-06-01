package generate

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/workflow"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func buildTestIR(
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
) (*pipeline.IR, error) {
	return buildTestIRWithApply(cfg, execCfg, contributions, depGraph, allModules, targetModules, true)
}

func buildTestIRWithApply(
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
	applyEnabled bool,
) (*pipeline.IR, error) {
	s := newSettings(cfg, execCfg)
	resourceRequests := []pipeline.ResourceRequest(nil)
	if !applyEnabled {
		resourceRequests = append(resourceRequests, pipeline.AllPlanResources(pipeline.ResourceKindPlanBinary))
	}
	intent, err := pipeline.NewBuildIntent(pipeline.BuildIntentOptions{
		ApplyEnabled:     applyEnabled,
		ResourceRequests: resourceRequests,
	})
	if err != nil {
		return nil, err
	}
	return pipeline.BuildProjectIR(pipeline.ProjectIRRequest{
		Project: &workflow.ProjectResult{
			Workflow: &workflow.Result{
				Filtered: workflow.NewModuleSet(allModules),
				Graph:    depGraph,
			},
			Targets: targetModules,
		},
		Script: pipeline.ScriptConfig{
			InitEnabled: s.initEnabled(),
			Env:         execCfg.Env,
		},
		Contributions: contributions,
		Intent:        intent,
	})
}
