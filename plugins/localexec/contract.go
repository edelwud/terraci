package localexec

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/plugin"
	localexecinternal "github.com/edelwud/terraci/plugins/localexec/internal"
	"github.com/edelwud/terraci/plugins/localexec/internal/flow"
)

// ExecutionMode determines which local pipeline shape to execute.
type ExecutionMode int

const (
	// ExecutionModeRun executes the full local flow, including apply jobs.
	ExecutionModeRun ExecutionMode = iota
	// ExecutionModePlan executes plan jobs and resource-dependent DAG jobs.
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

// Result describes one local execution invocation.
type Result struct {
	execution     *execution.Result
	summaryReport *ci.Report
	skipped       bool
	diagnostics   diagnostic.List
}

func newResult(result *flow.Result) *Result {
	if result == nil {
		return nil
	}
	return &Result{
		execution:     result.Execution(),
		summaryReport: result.SummaryReport(),
		skipped:       result.Skipped(),
		diagnostics:   result.Diagnostics(),
	}
}

// Execution returns the immutable execution result, if execution ran.
func (r *Result) Execution() *execution.Result {
	if r == nil {
		return nil
	}
	return r.execution.Clone()
}

// SummaryReport returns the aggregate local summary report, if one was loaded.
func (r *Result) SummaryReport() *ci.Report {
	if r == nil {
		return nil
	}
	return r.summaryReport.Clone()
}

// Skipped reports whether target selection resolved to no modules.
func (r *Result) Skipped() bool {
	return r != nil && r.skipped
}

// Diagnostics returns non-fatal diagnostics emitted during the invocation.
func (r *Result) Diagnostics() diagnostic.List {
	if r == nil {
		return diagnostic.List{}
	}
	return r.diagnostics
}

// Executor runs the local execution flow.
type Executor interface {
	Run(ctx context.Context, req ExecuteRequest) (*Result, error)
}

type ExecutorOption func(*executorOptions)

type executorOptions struct {
	eventSink execution.EventSink
}

func WithEventSink(sink execution.EventSink) ExecutorOption {
	return func(opts *executorOptions) {
		opts.eventSink = sink
	}
}

// NewExecutor constructs the public local-exec executor contract.
func NewExecutor(appCtx *plugin.AppContext, opts ...ExecutorOption) Executor {
	options := executorOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	internalOpts := make([]localexecinternal.Option, 0)
	if options.eventSink != nil {
		internalOpts = append(internalOpts, localexecinternal.WithEventSink(options.eventSink))
	}
	return executorAdapter{executor: localexecinternal.NewExecutor(appCtx, internalOpts...)}
}

type executorAdapter struct {
	executor *localexecinternal.Executor
}

func (e executorAdapter) Run(ctx context.Context, req ExecuteRequest) (*Result, error) {
	mapped, err := mapExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	result, err := e.executor.Run(ctx, mapped)
	return newResult(result), err
}

func mapExecuteRequest(req ExecuteRequest) (localexecinternal.Request, error) {
	switch req.Mode {
	case ExecutionModeRun, ExecutionModePlan:
	default:
		return localexecinternal.Request{}, fmt.Errorf("invalid local-exec mode %q", req.Mode.String())
	}

	mapped := localexecinternal.Request{
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
