package generate

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/terraformrun"
	"github.com/edelwud/terraci/pkg/workflow"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

func buildTestIRWithApply(
	cfg *configpkg.Config,
	profile terraformrun.Profile,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
	applyEnabled bool,
) (*pipeline.IR, error) {
	s := newSettings(cfg, profile)
	intent, err := buildIntentForApply(applyEnabled)
	if err != nil {
		return nil, err
	}
	terraformConfig, err := pipeline.NewTerraformJobConfig(pipeline.TerraformJobConfigOptions{
		Binary:      profile.Binary().String(),
		InitEnabled: s.initEnabled(),
		Env:         profile.Env(),
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
		Terraform:     terraformConfig,
		Contributions: contributions,
		Intent:        intent,
	})
}

func buildIntentForApply(applyEnabled bool) (pipeline.BuildIntent, error) {
	if applyEnabled {
		return pipeline.ApplyBuildIntent()
	}
	return pipeline.PlanBuildIntent(pipeline.AllPlanResources(pipeline.ResourceKindPlanBinary))
}
