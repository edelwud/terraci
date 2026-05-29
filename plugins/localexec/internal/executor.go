package internal

import (
	"context"

	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/localexec/internal/flow"
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
}

type Option = flow.Option

func WithEventSink(sink execution.EventSink) Option {
	return flow.WithEventSink(sink)
}

func NewExecutor(appCtx *plugin.AppContext, opts ...Option) *Executor {
	return &Executor{
		useCase: flow.New(appCtx, opts...),
	}
}

func (e *Executor) Run(ctx context.Context, req Request) (*flow.Result, error) {
	normalized, err := spec.NormalizeRequest(req)
	if err != nil {
		return nil, err
	}
	return e.useCase.Run(ctx, normalized)
}
