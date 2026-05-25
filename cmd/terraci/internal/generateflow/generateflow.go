// Package generateflow owns pipeline generation use-case orchestration.
package generateflow

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/cmd/terraci/internal/projectflow"
	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/pipeline"
)

// Runtime contains immutable dependencies needed to generate a pipeline.
type Runtime struct {
	prepared *runflow.Prepared
	project  projectflow.Runtime
}

// NewRuntime creates a generation runtime from prepared command state.
func NewRuntime(prepared *runflow.Prepared) Runtime {
	return Runtime{
		prepared: prepared,
		project:  projectflow.NewRuntime(prepared),
	}
}

// Request describes one pipeline generation request.
type Request struct {
	Filters     filter.Flags
	ChangedOnly bool
	BaseRef     string
	PlanOnly    bool
	DryRun      bool
}

// Result contains the pipeline generation outcome.
type Result struct {
	Project  *projectflow.Result
	Pipeline pipeline.GeneratedPipeline
	DryRun   *pipeline.DryRunResult
	Skipped  bool
}

// Run executes pipeline generation or dry-run preview.
func Run(ctx context.Context, runtime Runtime, req Request) (*Result, error) {
	project, err := projectflow.Run(ctx, runtime.project, projectflow.Request{
		Filters:       req.Filters,
		SelectTargets: true,
		ChangedOnly:   req.ChangedOnly,
		BaseRef:       req.BaseRef,
	})
	if err != nil {
		return nil, err
	}

	result := &Result{Project: project}
	if len(project.Targets) == 0 {
		result.Skipped = true
		return result, nil
	}

	generator, err := newPipelineGenerator(runtime, project, req.PlanOnly)
	if err != nil {
		return nil, err
	}
	if req.DryRun {
		dryRun, dryRunErr := generator.DryRun()
		if dryRunErr != nil {
			return nil, fmt.Errorf("dry run: %w", dryRunErr)
		}
		result.DryRun = dryRun
		return result, nil
	}

	generated, err := generator.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate pipeline: %w", err)
	}
	result.Pipeline = generated
	return result, nil
}

func newPipelineGenerator(runtime Runtime, project *projectflow.Result, planOnly bool) (pipeline.Generator, error) {
	appCtx := runtime.prepared.AppContext()
	provider, err := appCtx.CIResolver().ResolveCIProvider()
	if err != nil {
		return nil, fmt.Errorf("resolve CI provider: %w", err)
	}

	exec := execution.ConfigFromProject(runtime.prepared.Config())
	requirements := exec.BuildRequirements().Merge(provider.PipelineRequirements(appCtx))
	if planOnly {
		requirements = requirements.Merge(pipeline.BuildRequirements{PlanOnly: true})
	}
	ir, err := pipeline.BuildProjectIR(pipeline.ProjectIRRequest{
		Project:       project,
		Contributions: appCtx.PipelineContributions(),
		Requirements:  requirements,
		PlanEnabled:   exec.PlanEnabled,
		Script: pipeline.ScriptConfig{
			InitEnabled: exec.InitEnabled,
			PlanEnabled: exec.PlanEnabled,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build pipeline IR: %w", err)
	}
	return provider.NewGenerator(appCtx, ir), nil
}
