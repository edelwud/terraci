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

type defaultBuilder struct {
	appCtx *plugin.AppContext
}

func New(appCtx *plugin.AppContext) Builder {
	return defaultBuilder{appCtx: appCtx}
}

func (b defaultBuilder) Build(targets []*discovery.Module, result *workflow.Result, execCfg execution.Config, mode spec.ExecutionMode) (*execution.Plan, error) {
	contributions := registry.CollectContributions(b.appCtx)

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
