package localexec

import (
	"context"

	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/plugin"
	localexecinternal "github.com/edelwud/terraci/plugins/localexec/internal"
)

// ExecutionMode determines which pipeline phases to execute.
type ExecutionMode = localexecinternal.ExecutionMode

const (
	ExecutionModeFull      = localexecinternal.ExecutionModeFull
	ExecutionModePlanOnly  = localexecinternal.ExecutionModePlanOnly
	ExecutionModeApplyOnly = localexecinternal.ExecutionModeApplyOnly
)

// ExecuteRequest describes one local-exec invocation.
type ExecuteRequest struct {
	ChangedOnly bool
	BaseRef     string
	Mode        ExecutionMode
	ModulePath  string
	Parallelism int
	DryRun      bool
	Filters     *filter.Flags
}

// Executor runs the local execution flow.
type Executor interface {
	Run(ctx context.Context, req ExecuteRequest) error
}

// NewExecutor constructs the public local-exec executor contract.
func NewExecutor(appCtx *plugin.AppContext) Executor {
	return executorAdapter{executor: localexecinternal.NewExecutor(appCtx)}
}

type executorAdapter struct {
	executor *localexecinternal.Executor
}

func (e executorAdapter) Run(ctx context.Context, req ExecuteRequest) error {
	return e.executor.Run(ctx, localexecinternal.ExecuteRequest{
		ChangedOnly: req.ChangedOnly,
		BaseRef:     req.BaseRef,
		Mode:        req.Mode,
		ModulePath:  req.ModulePath,
		Parallelism: req.Parallelism,
		DryRun:      req.DryRun,
		Filters:     req.Filters,
	})
}
