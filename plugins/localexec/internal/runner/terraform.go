package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-exec/tfexec"

	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
)

type terraformRunner interface {
	RunPlan(ctx context.Context, job *pipeline.Job, op *pipeline.TerraformOperation) error
	RunApply(ctx context.Context, job *pipeline.Job, op *pipeline.TerraformOperation) error
}

type terraformOperationRunner struct {
	binaryPath string
	workspace  execution.Workspace
	execConfig execution.Config
}

func (r *terraformOperationRunner) RunPlan(ctx context.Context, job *pipeline.Job, op *pipeline.TerraformOperation) error {
	tf, err := tfexec.NewTerraform(r.workspace.ModuleDir(op.ModulePath), r.binaryPath)
	if err != nil {
		return fmt.Errorf("%s: create terraform runner: %w", job.Name, err)
	}
	if err = tf.SetEnv(mergeEnv(environMap(), r.execConfig.Env, job.Env)); err != nil {
		return fmt.Errorf("%s: set env: %w", job.Name, err)
	}

	if op.InitEnabled {
		if err = tf.Init(ctx); err != nil {
			return fmt.Errorf("%s: init: %w", job.Name, err)
		}
	}

	opts := []tfexec.PlanOption{tfexec.Out(filepath.Base(op.PlanFile))}
	if r.execConfig.Parallelism > 0 {
		opts = append(opts, tfexec.Parallelism(r.execConfig.Parallelism))
	}
	if _, err = tf.Plan(ctx, opts...); err != nil {
		return fmt.Errorf("%s: plan: %w", job.Name, err)
	}

	if !op.DetailedPlan {
		return nil
	}

	raw, err := tf.ShowPlanFileRaw(ctx, filepath.Base(op.PlanFile))
	if err != nil {
		return fmt.Errorf("%s: show plan text: %w", job.Name, err)
	}
	if err = os.WriteFile(r.workspace.PlanTextFile(op.ModulePath), []byte(raw), 0o600); err != nil {
		return fmt.Errorf("%s: write plan.txt: %w", job.Name, err)
	}

	planJSON, err := tf.ShowPlanFile(ctx, filepath.Base(op.PlanFile))
	if err != nil {
		return fmt.Errorf("%s: show plan json: %w", job.Name, err)
	}
	data, err := json.MarshalIndent(planJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("%s: marshal plan.json: %w", job.Name, err)
	}
	if err = os.WriteFile(r.workspace.PlanJSONFile(op.ModulePath), data, 0o600); err != nil {
		return fmt.Errorf("%s: write plan.json: %w", job.Name, err)
	}

	return nil
}

func (r *terraformOperationRunner) RunApply(ctx context.Context, job *pipeline.Job, op *pipeline.TerraformOperation) error {
	tf, err := tfexec.NewTerraform(r.workspace.ModuleDir(op.ModulePath), r.binaryPath)
	if err != nil {
		return fmt.Errorf("%s: create terraform runner: %w", job.Name, err)
	}
	if err = tf.SetEnv(mergeEnv(environMap(), r.execConfig.Env, job.Env)); err != nil {
		return fmt.Errorf("%s: set env: %w", job.Name, err)
	}

	if op.InitEnabled {
		if err = tf.Init(ctx); err != nil {
			return fmt.Errorf("%s: init: %w", job.Name, err)
		}
	}

	var opts []tfexec.ApplyOption
	if op.UsePlanFile {
		opts = append(opts, tfexec.DirOrPlan(filepath.Base(op.PlanFile)))
	}
	if err := tf.Apply(ctx, opts...); err != nil {
		return fmt.Errorf("%s: apply: %w", job.Name, err)
	}
	return nil
}
