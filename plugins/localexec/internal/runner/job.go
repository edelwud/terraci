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

type jobRunner struct {
	main     operationRunner
	commands commandRunner
}

func (r *jobRunner) Run(ctx context.Context, job *pipeline.Job) error {
	if job == nil {
		return errors.New("job is nil")
	}
	if job.Module == nil {
		return r.runStandaloneJob(ctx, job)
	}
	if r.main == nil {
		return errors.New("operation runner is not configured")
	}
	return r.main.Run(ctx, job)
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
