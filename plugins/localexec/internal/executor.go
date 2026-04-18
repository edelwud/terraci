package internal

import (
	"context"

	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/plugin"
)

// ExecutionMode determines which pipeline phases to execute.
type ExecutionMode int

const (
	// ExecutionModeFull runs all phases (plan + apply).
	ExecutionModeFull ExecutionMode = iota
	// ExecutionModePlanOnly runs pre-plan, plan, and post-plan phases.
	ExecutionModePlanOnly
	// ExecutionModeApplyOnly runs pre-apply, apply, and post-apply phases.
	ExecutionModeApplyOnly
)

type ExecuteRequest struct {
	ChangedOnly bool
	BaseRef     string
	Mode        ExecutionMode
	ModulePath  string
	Parallelism int
	DryRun      bool
	Filters     *filter.Flags
}

type Executor struct {
	useCase *localExecUseCase
}

func NewExecutor(appCtx *plugin.AppContext) *Executor {
	return &Executor{
		useCase: newLocalExecUseCase(appCtx),
	}
}

func (e *Executor) Run(ctx context.Context, req ExecuteRequest) error {
	return e.useCase.Run(ctx, executeRequest{
		changedOnly: req.ChangedOnly,
		baseRef:     req.BaseRef,
		mode:        req.Mode,
		modulePath:  req.ModulePath,
		parallelism: req.Parallelism,
		dryRun:      req.DryRun,
		filters:     req.Filters,
	})
}
