package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/pkg/workflow"
)

type executeRequest struct {
	changedOnly bool
	baseRef     string
	mode        ExecutionMode
	modulePath  string
	parallelism int
	dryRun      bool
	filters     *filter.Flags
}

type localExecUseCase struct {
	appCtx         *plugin.AppContext
	runtimeFactory runtimeFactory
	output         executionOutput
}

func newLocalExecUseCase(appCtx *plugin.AppContext) *localExecUseCase {
	return &localExecUseCase{
		appCtx:         appCtx,
		runtimeFactory: newRuntimeFactory(),
		output:         logOutput{serviceDir: appCtx.ServiceDir()},
	}
}

func (u *localExecUseCase) Run(ctx context.Context, req executeRequest) error {
	result, err := workflow.Run(ctx, workflowOptionsFromContext(u.appCtx, req.filters))
	if err != nil {
		return err
	}

	targets, err := u.resolveTargets(ctx, req, result)
	if err != nil {
		return err
	}

	execRuntime, err := u.runtimeFactory.Build(u.appCtx, runtimeOptions{parallelism: req.parallelism})
	if err != nil {
		return err
	}

	plan, err := u.buildExecutionPlan(targets, result, execRuntime.execConfig, req.mode)
	if err != nil {
		return err
	}

	if req.dryRun {
		return u.output.DryRun(plan)
	}

	reporter := progressReporter{}
	resultExec, err := execution.NewExecutor(
		execRuntime.JobRunner(),
		execution.WithParallelism(execRuntime.execConfig.Parallelism),
		execution.WithEventSink(reporter),
	).Execute(ctx, plan)
	if err != nil {
		return u.output.Failure(resultExec, err)
	}

	u.output.Completed(resultExec)
	return nil
}

func (u *localExecUseCase) resolveTargets(ctx context.Context, req executeRequest, result *workflow.Result) ([]*discovery.Module, error) {
	targets := result.FilteredModules
	if req.modulePath != "" {
		targets = filterModules(targets, req.modulePath)
	}
	if req.changedOnly {
		var err error
		targets, err = detectChangedTargetModules(ctx, u.appCtx, req.baseRef, result.FullIndex, result.FilteredIndex, result.Graph)
		if err != nil {
			return nil, err
		}
	}
	if len(targets) == 0 {
		return nil, errors.New("no modules remaining after filtering")
	}
	return targets, nil
}

func (u *localExecUseCase) buildExecutionPlan(targets []*discovery.Module, result *workflow.Result, execCfg execution.Config, mode ExecutionMode) (*execution.Plan, error) {
	contributions := registry.CollectContributions(u.appCtx)

	planOnly := mode == ExecutionModePlanOnly
	applyOnly := mode == ExecutionModeApplyOnly

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
			PlanEnabled:  execCfg.PlanEnabled && !applyOnly,
			AutoApprove:  applyOnly,
			DetailedPlan: execCfg.PlanMode == execution.PlanModeDetailed,
		},
		Contributions: contributions,
		PlanEnabled:   execCfg.PlanEnabled && !applyOnly,
		PlanOnly:      planOnly,
		ApplyOnly:     applyOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("build local execution plan: %w", err)
	}

	return execution.NewPlan(ir), nil
}

func workflowOptionsFromContext(appCtx *plugin.AppContext, ff *filter.Flags) workflow.Options {
	cfg := appCtx.Config()
	opts := ff.Merge(cfg.Exclude, cfg.Include)
	return workflow.Options{
		WorkDir:        appCtx.WorkDir(),
		Segments:       cfg.Structure.Segments,
		Excludes:       opts.Excludes,
		Includes:       opts.Includes,
		SegmentFilters: opts.Segments,
	}
}

func filterModules(modules []*discovery.Module, modulePath string) []*discovery.Module {
	if modulePath == "" {
		return modules
	}
	filtered := modules[:0]
	for _, module := range modules {
		if module.RelativePath == modulePath {
			filtered = append(filtered, module)
		}
	}
	return filtered
}

func detectChangedTargetModules(
	ctx context.Context,
	appCtx *plugin.AppContext,
	baseRef string,
	fullIndex, filteredIndex *discovery.ModuleIndex,
	depGraph *graph.DependencyGraph,
) ([]*discovery.Module, error) {
	detector, err := registry.ResolveChangeDetector()
	if err != nil {
		return nil, fmt.Errorf("change detection: %w", err)
	}
	changedModules, _, err := detector.DetectChangedModules(ctx, appCtx, baseRef, fullIndex)
	if err != nil {
		return nil, err
	}
	affectedIDs := depGraph.GetAffectedModules(moduleIDs(changedModules))
	var targets []*discovery.Module
	for _, id := range affectedIDs {
		if mod := filteredIndex.ByID(id); mod != nil {
			targets = append(targets, mod)
			continue
		}
		if mod := fullIndex.ByID(id); mod != nil {
			targets = append(targets, mod)
		}
	}
	return targets, nil
}

func moduleIDs(modules []*discovery.Module) []string {
	ids := make([]string, len(modules))
	for i, module := range modules {
		ids[i] = module.ID()
	}
	return ids
}
