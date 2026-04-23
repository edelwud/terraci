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
	switch job.Operation.Type {
	case pipeline.OperationTypeTerraformPlan:
		return r.terraform.RunPlan(ctx, job, job.Operation.Terraform)
	case pipeline.OperationTypeTerraformApply:
		return r.terraform.RunApply(ctx, job, job.Operation.Terraform)
	case pipeline.OperationTypeCommands:
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
	if job != nil && job.Type == pipeline.JobTypeApply {
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
