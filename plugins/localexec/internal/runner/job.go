package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/pipeline"
)

type operationRunner interface {
	Run(ctx context.Context, job *pipeline.Job) error
}

type operationDispatcher struct {
	terraform terraformRunner
	commands  commandRunner
}

func (r operationDispatcher) Run(ctx context.Context, job *pipeline.Job) error {
	if job == nil {
		return errors.New("job is nil")
	}

	switch job.Operation.Type {
	case pipeline.OperationTypeTerraformPlan:
		if r.terraform == nil {
			return errors.New("terraform runner is not configured")
		}
		if job.Operation.Terraform == nil {
			return fmt.Errorf("%s: terraform plan operation is nil", job.Name)
		}
		return r.terraform.RunPlan(ctx, job, job.Operation.Terraform)
	case pipeline.OperationTypeTerraformApply:
		if r.terraform == nil {
			return errors.New("terraform runner is not configured")
		}
		if job.Operation.Terraform == nil {
			return fmt.Errorf("%s: terraform apply operation is nil", job.Name)
		}
		return r.terraform.RunApply(ctx, job, job.Operation.Terraform)
	case pipeline.OperationTypeCommands:
		if r.commands == nil {
			return errors.New("command runner is not configured")
		}
		for _, command := range job.Operation.Commands {
			if err := r.commands.Run(ctx, commandSpec{
				JobName:      job.Name,
				Command:      command,
				Env:          job.Env,
				AllowFailure: job.AllowFailure,
			}); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported operation type %q", job.Operation.Type)
	}
}

type phaseRunner struct {
	commands commandRunner
	main     operationRunner
}

func (r phaseRunner) Run(ctx context.Context, job *pipeline.Job) error {
	if r.commands == nil {
		return errors.New("command runner is not configured")
	}
	if r.main == nil {
		return errors.New("operation runner is not configured")
	}

	prePhase, postPhase := phasesForJob(job)
	for _, step := range job.Steps {
		if step.Phase != prePhase {
			continue
		}
		if err := r.commands.Run(ctx, commandSpec{
			JobName:    job.Name,
			ModulePath: modulePath(job),
			Command:    step.Command,
			Env:        job.Env,
		}); err != nil {
			return err
		}
	}
	if err := r.main.Run(ctx, job); err != nil {
		return err
	}
	for _, step := range job.Steps {
		if step.Phase != postPhase {
			continue
		}
		if err := r.commands.Run(ctx, commandSpec{
			JobName:    job.Name,
			ModulePath: modulePath(job),
			Command:    step.Command,
			Env:        job.Env,
		}); err != nil {
			return err
		}
	}
	return nil
}

type jobRunner struct {
	phaseRunner phaseRunner
	commands    commandRunner
}

func (r *jobRunner) Run(ctx context.Context, job *pipeline.Job) error {
	if job == nil {
		return errors.New("job is nil")
	}
	if job.Module == nil {
		return r.runStandaloneJob(ctx, job)
	}
	return r.phaseRunner.Run(ctx, job)
}

func (r *jobRunner) runStandaloneJob(ctx context.Context, job *pipeline.Job) error {
	if r.commands == nil {
		return errors.New("command runner is not configured")
	}
	for _, command := range job.Operation.Commands {
		if err := r.commands.Run(ctx, commandSpec{
			JobName:      job.Name,
			Command:      command,
			Env:          job.Env,
			AllowFailure: job.AllowFailure,
		}); err != nil {
			return err
		}
	}
	return nil
}

func phasesForJob(job *pipeline.Job) (pre, post pipeline.Phase) {
	// Branch on the operation payload, not on a parallel "JobType" field —
	// the latter defaulted to "plan" for contributed jobs and silently
	// captured them in plan-phase logic.
	if job != nil && job.Operation.Type == pipeline.OperationTypeTerraformApply {
		return pipeline.PhasePreApply, pipeline.PhasePostApply
	}
	return pipeline.PhasePrePlan, pipeline.PhasePostPlan
}

func modulePath(job *pipeline.Job) string {
	if job == nil || job.Module == nil {
		return ""
	}
	return job.Module.RelativePath
}
