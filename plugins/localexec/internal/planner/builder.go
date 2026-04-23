package planner

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

type Builder interface {
	Build(targets []*discovery.Module, result *workflow.Result, execCfg execution.Config, mode spec.ExecutionMode) (*execution.Plan, error)
}

type ContributionCollector interface {
	Collect(appCtx *plugin.AppContext) []*pipeline.Contribution
}

type defaultBuilder struct {
	appCtx        *plugin.AppContext
	contributions ContributionCollector
}

func New(appCtx *plugin.AppContext) Builder {
	return NewWithContributionCollector(appCtx, registryContributionCollector{})
}

func NewWithContributionCollector(appCtx *plugin.AppContext, collector ContributionCollector) Builder {
	if collector == nil {
		collector = registryContributionCollector{}
	}
	return defaultBuilder{appCtx: appCtx, contributions: collector}
}

func (b defaultBuilder) Build(targets []*discovery.Module, result *workflow.Result, execCfg execution.Config, mode spec.ExecutionMode) (*execution.Plan, error) {
	contributions := b.contributions.Collect(b.appCtx)

	planOnly := mode == spec.ExecutionModePlan
	if planOnly {
		execCfg.PlanEnabled = true
	}

	ir, err := pipeline.Build(pipeline.BuildOptions{
		DepGraph:      result.Graph,
		TargetModules: targets,
		AllModules:    result.FilteredModules,
		ModuleIndex:   result.FilteredIndex,
		Script: pipeline.ScriptConfig{
			InitEnabled:  execCfg.InitEnabled,
			PlanEnabled:  execCfg.PlanEnabled,
			DetailedPlan: execCfg.PlanMode == execution.PlanModeDetailed,
		},
		Contributions: contributions,
		PlanEnabled:   execCfg.PlanEnabled,
		PlanOnly:      planOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("build local execution plan: %w", err)
	}

	return execution.NewPlan(ir), nil
}

type registryContributionCollector struct{}

func (registryContributionCollector) Collect(appCtx *plugin.AppContext) []*pipeline.Contribution {
	return registry.CollectContributions(appCtx)
}
