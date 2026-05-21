// Package validateflow owns project validation orchestration.
package validateflow

import (
	"context"

	"github.com/edelwud/terraci/cmd/terraci/internal/projectflow"
	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
)

// Runtime contains immutable dependencies needed to validate a project.
type Runtime struct {
	project projectflow.Runtime
}

// NewRuntime creates a validation runtime from prepared command state.
func NewRuntime(prepared *runflow.Prepared) Runtime {
	return Runtime{project: projectflow.NewRuntime(prepared)}
}

// Request describes one validation request.
type Request struct {
	Filters filter.Flags
}

// Result contains validation diagnostics.
type Result struct {
	Project              *projectflow.Result
	DependencyLinks      int
	Cycles               [][]string
	Stats                graph.Stats
	ExecutionLevels      [][]string
	ExecutionLevelsError error
	Passed               bool
}

// Run validates project dependency graph health.
func Run(ctx context.Context, runtime Runtime, req Request) (*Result, error) {
	project, err := projectflow.Run(ctx, runtime.project, projectflow.Request{Filters: req.Filters})
	if err != nil {
		return nil, err
	}
	return Evaluate(project), nil
}

// Evaluate derives validation diagnostics from a discovered project.
func Evaluate(project *projectflow.Result) *Result {
	if project == nil || project.Workflow == nil || project.Workflow.Graph == nil {
		return &Result{Project: project}
	}
	result := &Result{
		Project: project,
		Stats:   project.Workflow.Graph.GetStats(),
	}
	for _, deps := range project.Workflow.Dependencies {
		result.DependencyLinks += len(deps.DependsOn)
	}

	result.Cycles = project.Workflow.Graph.DetectCycles()
	levels, err := project.Workflow.Graph.ExecutionLevels()
	if err != nil {
		result.ExecutionLevelsError = err
	} else {
		result.ExecutionLevels = levels
	}
	result.Passed = len(result.Cycles) == 0 && result.ExecutionLevelsError == nil
	return result
}
