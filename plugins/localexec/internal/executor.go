package internal

import (
	"context"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/localexec/internal/flow"
	"github.com/edelwud/terraci/plugins/localexec/internal/render"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

// ExecutionMode determines which local pipeline shape to execute.
type ExecutionMode = spec.ExecutionMode

const (
	ExecutionModeRun  = spec.ExecutionModeRun
	ExecutionModePlan = spec.ExecutionModePlan
)

type Request = spec.Request

type Executor struct {
	useCase *flow.UseCase
	output  render.Output
}

func NewExecutor(appCtx *plugin.AppContext) *Executor {
	return &Executor{
		useCase: flow.New(appCtx),
		output:  render.NewLogOutput(),
	}
}

func (e *Executor) Run(ctx context.Context, req Request) error {
	normalized, err := spec.NormalizeRequest(req)
	if err != nil {
		return err
	}
	result, err := e.useCase.Run(ctx, normalized)
	if err != nil {
		if result != nil {
			return e.output.Failure(result.Execution, err)
		}
		return e.output.Failure(nil, err)
	}
	if result == nil || result.Skipped {
		return nil
	}
	return e.output.Completed(result.Execution, result.SummaryReport)
}
