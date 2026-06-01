// Package generateflow owns pipeline generation use-case orchestration.
package generateflow

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/cmd/terraci/internal/projectflow"
	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/terraformrun"
)

// Runtime contains immutable dependencies needed to generate a pipeline.
type Runtime struct {
	prepared *runflow.Prepared
	project  projectflow.Runtime
}

// GenerateMode declares whether generation should include apply jobs.
type GenerateMode string

const (
	GenerateModeApply GenerateMode = "apply"
	GenerateModePlan  GenerateMode = "plan"
)

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
	Mode        GenerateMode
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

	mode := req.Mode
	if mode == "" {
		mode = GenerateModeApply
	}
	generator, err := newPipelineGenerator(runtime, project, mode)
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

func newPipelineGenerator(runtime Runtime, project *projectflow.Result, mode GenerateMode) (pipeline.Generator, error) {
	appCtx := runtime.prepared.AppContext()
	provider, err := appCtx.CIResolver().ResolveCIProvider()
	if err != nil {
		return nil, fmt.Errorf("resolve CI provider: %w", err)
	}

	profile, err := terraformrun.ProfileFromConfig(runtime.prepared.Config())
	if err != nil {
		return nil, fmt.Errorf("terraform profile: %w", err)
	}
	intent, err := intentForMode(mode)
	if err != nil {
		return nil, fmt.Errorf("build pipeline intent: %w", err)
	}
	terraformConfig, err := pipeline.NewTerraformJobConfig(pipeline.TerraformJobConfigOptions{
		Binary:      profile.Binary().String(),
		InitEnabled: profile.InitEnabled(),
		Env:         profile.Env(),
	})
	if err != nil {
		return nil, fmt.Errorf("terraform job config: %w", err)
	}
	ir, err := pipeline.BuildProjectIR(pipeline.ProjectIRRequest{
		Project:       project,
		Contributions: appCtx.PipelineContributions(),
		Intent:        intent,
		Terraform:     terraformConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("build pipeline IR: %w", err)
	}
	generator, err := provider.NewGenerator(appCtx, ir)
	if err != nil {
		return nil, fmt.Errorf("create CI generator: %w", err)
	}
	return generator, nil
}

func intentForMode(mode GenerateMode) (pipeline.BuildIntent, error) {
	switch mode {
	case "", GenerateModeApply:
		return pipeline.ApplyBuildIntent()
	case GenerateModePlan:
		return pipeline.PlanBuildIntent(pipeline.AllPlanResources(pipeline.ResourceKindPlanBinary))
	default:
		var zero pipeline.BuildIntent
		return zero, fmt.Errorf("unsupported generate mode %q", mode)
	}
}
