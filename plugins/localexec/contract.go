package localexec

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/plugin"
	localexecinternal "github.com/edelwud/terraci/plugins/localexec/internal"
)

// ExecutionMode determines which pipeline phases to execute.
type ExecutionMode int

const (
	// ExecutionModeRun executes the full local flow, including apply jobs.
	ExecutionModeRun ExecutionMode = iota
	// ExecutionModePlan executes plan and finalize phases only.
	ExecutionModePlan
)

// CLI subcommand names exposed by local-exec. Centralized so cobra
// definitions, mode strings, and tests share one source of truth.
const (
	cmdRun  = "run"
	cmdPlan = "plan"
)

func (m ExecutionMode) String() string {
	switch m {
	case ExecutionModeRun:
		return cmdRun
	case ExecutionModePlan:
		return cmdPlan
	default:
		return fmt.Sprintf("ExecutionMode(%d)", m)
	}
}

// ExecuteRequest describes one local-exec invocation.
type ExecuteRequest struct {
	// ChangedOnly narrows execution to changed modules and their dependents.
	ChangedOnly bool
	// BaseRef controls the comparison base for change detection.
	BaseRef string
	// Mode must be either ExecutionModeRun or ExecutionModePlan.
	Mode ExecutionMode
	// ModulePath selects a single module after filter resolution when set.
	ModulePath string
	// Parallelism <= 0 keeps the project execution config default.
	Parallelism int
	// Filters may be nil and is normalized to an empty filter set.
	Filters *filter.Flags
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
	mapped, err := mapExecuteRequest(req)
	if err != nil {
		return err
	}
	return e.executor.Run(ctx, mapped)
}

func mapExecuteRequest(req ExecuteRequest) (localexecinternal.ExecuteRequest, error) {
	switch req.Mode {
	case ExecutionModeRun, ExecutionModePlan:
	default:
		return localexecinternal.ExecuteRequest{}, fmt.Errorf("invalid local-exec mode %q", req.Mode.String())
	}

	mapped := localexecinternal.ExecuteRequest{
		ChangedOnly: req.ChangedOnly,
		BaseRef:     req.BaseRef,
		ModulePath:  req.ModulePath,
		Parallelism: req.Parallelism,
		Filters:     req.Filters,
	}
	if mapped.Filters == nil {
		mapped.Filters = &filter.Flags{}
	}

	switch req.Mode {
	case ExecutionModeRun:
		mapped.Mode = localexecinternal.ExecutionModeRun
	case ExecutionModePlan:
		mapped.Mode = localexecinternal.ExecutionModePlan
	}

	return mapped, nil
}
