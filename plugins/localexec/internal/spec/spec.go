package spec

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/filter"
)

// ExecutionMode determines which local pipeline shape to execute.
type ExecutionMode int

const (
	// ExecutionModeRun runs the full DAG, including apply jobs.
	ExecutionModeRun ExecutionMode = iota
	// ExecutionModePlan runs the plan-only DAG.
	ExecutionModePlan
)

func (m ExecutionMode) String() string {
	switch m {
	case ExecutionModeRun:
		return "run"
	case ExecutionModePlan:
		return "plan"
	default:
		return fmt.Sprintf("ExecutionMode(%d)", m)
	}
}

// ExecuteRequest describes one local-exec invocation.
type ExecuteRequest struct {
	ChangedOnly bool
	BaseRef     string
	Mode        ExecutionMode
	ModulePath  string
	Parallelism int
	Filters     *filter.Flags
}

// NormalizeExecuteRequest validates boundary semantics and fills safe defaults.
//
// Filters defaults to an empty filter set. Parallelism <= 0 keeps the project
// execution config default and is intentionally not rewritten here.
func NormalizeExecuteRequest(req ExecuteRequest) (ExecuteRequest, error) {
	if req.Filters == nil {
		req.Filters = &filter.Flags{}
	}

	switch req.Mode {
	case ExecutionModeRun, ExecutionModePlan:
		return req, nil
	default:
		return ExecuteRequest{}, fmt.Errorf("invalid local-exec mode %q", req.Mode.String())
	}
}
