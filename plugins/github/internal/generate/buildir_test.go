package generate

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/workflow"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

func buildTestIRWithApply(
	_ *configpkg.Config,
	terraformConfigOptions pipeline.TerraformJobConfigOptions,
	contributions pipeline.ContributionSet,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
	applyEnabled bool,
) (*pipeline.IR, error) {
	intent, err := buildIntentForApply(applyEnabled)
	if err != nil {
		return nil, err
	}
	if terraformConfigOptions.Binary == "" {
		terraformConfigOptions.Binary = "terraform"
	}
	terraformConfig, err := pipeline.NewTerraformJobConfig(terraformConfigOptions)
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
