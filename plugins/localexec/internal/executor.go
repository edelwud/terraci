package internal

import (
	"context"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/localexec/internal/flow"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

// ExecutionMode determines which pipeline phases to execute.
type ExecutionMode = spec.ExecutionMode

const (
	ExecutionModeRun  = spec.ExecutionModeRun
	ExecutionModePlan = spec.ExecutionModePlan
)

type ExecuteRequest = spec.ExecuteRequest

type Executor struct {
	useCase *flow.UseCase
}

func NewExecutor(appCtx *plugin.AppContext) *Executor {
	return &Executor{
		useCase: flow.New(appCtx),
	}
}

func (e *Executor) Run(ctx context.Context, req ExecuteRequest) error {
	normalized, err := spec.NormalizeExecuteRequest(req)
	if err != nil {
		return err
	}
	return e.useCase.Run(ctx, normalized)
}
