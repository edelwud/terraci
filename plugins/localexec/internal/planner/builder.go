package planner

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

type Builder interface {
	Build(targets []*discovery.Module, result *workflow.Result, execCfg execution.Config, mode spec.ExecutionMode, contributions []*pipeline.Contribution) (*pipeline.IR, error)
}

type defaultBuilder struct{}

func New() Builder {
	return defaultBuilder{}
}

func (defaultBuilder) Build(targets []*discovery.Module, result *workflow.Result, execCfg execution.Config, mode spec.ExecutionMode, contributions []*pipeline.Contribution) (*pipeline.IR, error) {
	planOnly := mode == spec.ExecutionModePlan
	if planOnly {
		execCfg.PlanEnabled = true
	}
	requirements := execCfg.BuildRequirements().Merge(pipeline.BuildRequirements{PlanOnly: planOnly})

	ir, err := pipeline.Build(pipeline.BuildOptions{
		DepGraph:      result.Graph,
		TargetModules: targets,
		AllModules:    result.Filtered.Modules,
		ModuleIndex:   result.Filtered.Index,
		Script: pipeline.ScriptConfig{
			InitEnabled: execCfg.InitEnabled,
			PlanEnabled: execCfg.PlanEnabled,
		},
		Contributions: contributions,
		Requirements:  requirements,
		PlanEnabled:   execCfg.PlanEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("build local execution plan: %w", err)
	}

	return ir, nil
}
