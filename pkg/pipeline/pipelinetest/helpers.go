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
	ir := MustCommandIR(tb, opts)
	jobs := ir.Jobs()
	for i := range jobs {
		if jobs[i].Name() == opts.Name {
			return jobs[i]
		}
	}
	tb.Fatalf("command job %q not found", opts.Name)
	var zero pipeline.Job
	return zero
}

// MustCommandIR builds a valid command-only IR for tests.
func MustCommandIR(tb testing.TB, opts ...pipeline.ContributedJobOptions) *pipeline.IR {
	tb.Helper()
	jobs := make([]pipeline.ContributedJob, 0, len(opts))
	for _, opt := range opts {
		job, err := pipeline.NewContributedJob(opt)
		if err != nil {
			tb.Fatalf("NewContributedJob() error = %v", err)
		}
		jobs = append(jobs, job)
	}
	var contributions []*pipeline.Contribution
	if len(jobs) > 0 {
		contribution, err := pipeline.NewContribution(jobs...)
		if err != nil {
			tb.Fatalf("NewContribution() error = %v", err)
		}
		contributions = []*pipeline.Contribution{contribution}
	}
	ir, err := pipeline.BuildProjectIR(pipeline.ProjectIRRequest{
		Project:       emptyProject(),
		Contributions: contributions,
	})
	if err != nil {
		tb.Fatalf("BuildProjectIR() error = %v", err)
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
	var zero pipeline.Job
	return zero
}

func emptyProject() *workflow.ProjectResult {
	return &workflow.ProjectResult{
		Workflow: &workflow.Result{
			Filtered: workflow.NewModuleSet(nil),
			Graph:    graph.NewDependencyGraph(),
		},
		Targets: []*discovery.Module{},
	}
}
