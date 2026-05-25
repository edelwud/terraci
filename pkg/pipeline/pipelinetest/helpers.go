// Package pipelinetest provides public helpers for tests that need valid
// pipeline value objects without constructing IR internals directly.
package pipelinetest

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/workflow"
)

// MustCommandJob builds a valid command job for tests.
func MustCommandJob(tb testing.TB, opts pipeline.ContributedJobOptions) pipeline.Job {
	tb.Helper()
	job, err := pipeline.NewContributedJob(opts)
	if err != nil {
		tb.Fatalf("NewContributedJob() error = %v", err)
	}
	return pipeline.NewCommandJob(job)
}

// MustCommandIR builds a valid command-only IR for tests.
func MustCommandIR(tb testing.TB, opts ...pipeline.ContributedJobOptions) *pipeline.IR {
	tb.Helper()
	jobs := make([]pipeline.Job, 0, len(opts))
	for _, opt := range opts {
		jobs = append(jobs, MustCommandJob(tb, opt))
	}
	ir, err := pipeline.NewIR(jobs...)
	if err != nil {
		tb.Fatalf("NewIR() error = %v", err)
	}
	return ir
}

// MustSingleModuleIR builds a valid plan/apply IR for a single module.
func MustSingleModuleIR(tb testing.TB, module *discovery.Module) *pipeline.IR {
	tb.Helper()
	depGraph := graph.NewDependencyGraph()
	depGraph.AddNode(module)
	ir, err := pipeline.BuildProjectIR(pipeline.ProjectIRRequest{
		Project: &workflow.ProjectResult{
			Workflow: &workflow.Result{
				Filtered: workflow.NewModuleSet([]*discovery.Module{module}),
				Graph:    depGraph,
			},
		},
		Script:      pipeline.ScriptConfig{InitEnabled: true, PlanEnabled: true},
		PlanEnabled: true,
	})
	if err != nil {
		tb.Fatalf("BuildProjectIR() error = %v", err)
	}
	return ir
}

// MustJobByKind returns the first job with kind.
func MustJobByKind(tb testing.TB, ir *pipeline.IR, kind pipeline.JobKind) pipeline.Job {
	tb.Helper()
	jobs := ir.Jobs()
	for i := range jobs {
		if jobs[i].Kind() == kind {
			return jobs[i]
		}
	}
	tb.Fatalf("job kind %q not found", kind)
	return pipeline.Job{}
}
