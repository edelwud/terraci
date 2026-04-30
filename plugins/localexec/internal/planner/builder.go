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
	Build(targets []*discovery.Module, result *workflow.Result, execCfg execution.Config, mode spec.ExecutionMode, contributions []*pipeline.Contribution) (*execution.Plan, error)
}

type defaultBuilder struct{}

func New() Builder {
	return defaultBuilder{}
}

func (defaultBuilder) Build(targets []*discovery.Module, result *workflow.Result, execCfg execution.Config, mode spec.ExecutionMode, contributions []*pipeline.Contribution) (*execution.Plan, error) {
	planOnly := mode == spec.ExecutionModePlan
	if planOnly {
		execCfg.PlanEnabled = true
		contributions = planModeContributions(contributions)
	}

	ir, err := pipeline.Build(pipeline.BuildOptions{
		DepGraph:      result.Graph,
		TargetModules: targets,
		AllModules:    result.Filtered.Modules,
		ModuleIndex:   result.Filtered.Index,
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

func planModeContributions(contributions []*pipeline.Contribution) []*pipeline.Contribution {
	if len(contributions) == 0 {
		return nil
	}

	filtered := make([]*pipeline.Contribution, 0, len(contributions))
	for _, contribution := range contributions {
		if contribution == nil {
			continue
		}
		next := &pipeline.Contribution{
			Steps: append([]pipeline.Step(nil), contribution.Steps...),
		}
		for _, job := range contribution.Jobs {
			if isPlanModeJobPhase(job.Phase) {
				next.Jobs = append(next.Jobs, job)
			}
		}
		if len(next.Steps) > 0 || len(next.Jobs) > 0 {
			filtered = append(filtered, next)
		}
	}

	return filtered
}

func isPlanModeJobPhase(phase pipeline.Phase) bool {
	switch phase {
	case pipeline.PhasePrePlan, pipeline.PhasePostPlan, pipeline.PhaseFinalize:
		return true
	case pipeline.PhasePreApply, pipeline.PhasePostApply:
		return false
	default:
		return false
	}
}
