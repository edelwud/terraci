package domain

import (
	"testing"
)

func TestWorkflowGettersReturnDefensiveCopies(t *testing.T) {
	step := NewStep(StepOptions{
		Name: "Plan",
		Run:  "terraform plan",
		Env:  map[string]string{"STEP": "true"},
	})
	job, err := NewJob(JobOptions{
		Name:   "plan-vpc",
		RunsOn: "ubuntu-latest",
		Needs:  []string{"setup"},
		Env:    map[string]string{"TF_MODULE": "vpc"},
		Steps:  []Step{step},
	})
	if err != nil {
		t.Fatal(err)
	}
	builder := NewWorkflowBuilder(WorkflowOptions{
		Name: "Terraform",
		Env:  map[string]string{"GLOBAL": "true"},
	})
	mustAddWorkflowJob(t, builder, "plan-vpc", job)
	workflow := mustBuildWorkflow(t, builder)

	got, ok := workflow.Job("plan-vpc")
	if !ok {
		t.Fatal("plan-vpc job not found")
	}
	needs := got.Needs()
	needs[0] = "changed"
	if !got.HasNeed("setup") {
		t.Fatalf("Job.Needs() leaked mutation: %#v", got.Needs())
	}
	env := got.Env()
	env["TF_MODULE"] = "changed"
	if got.Env()["TF_MODULE"] != "vpc" {
		t.Fatalf("Job.Env() leaked mutation: %#v", got.Env())
	}
	steps := got.Steps()
	stepEnv := steps[0].Env()
	stepEnv["STEP"] = "changed"
	if got.Steps()[0].Env()["STEP"] != "true" {
		t.Fatalf("Step.Env() leaked mutation: %#v", got.Steps()[0].Env())
	}
	workflowEnv := workflow.Env()
	workflowEnv["GLOBAL"] = "changed"
	if workflow.Env()["GLOBAL"] != "true" {
		t.Fatalf("Workflow.Env() leaked mutation: %#v", workflow.Env())
	}
}

func TestWorkflowBuilderValidatesDuplicateJobs(t *testing.T) {
	step := NewStep(StepOptions{Name: "Checkout", Uses: "actions/checkout@v4"})
	job, err := NewJob(JobOptions{RunsOn: "ubuntu-latest", Steps: []Step{step}})
	if err != nil {
		t.Fatal(err)
	}
	builder := NewWorkflowBuilder(WorkflowOptions{})
	if err := builder.AddJob("test", job); err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}
	if err := builder.AddJob("test", job); err == nil {
		t.Fatal("AddJob() error = nil, want duplicate job error")
	}
}

func mustAddWorkflowJob(tb testing.TB, builder *WorkflowBuilder, name string, job Job) {
	tb.Helper()
	if err := builder.AddJob(name, job); err != nil {
		tb.Fatalf("AddJob(%q) error = %v", name, err)
	}
}

func mustBuildWorkflow(tb testing.TB, builder *WorkflowBuilder) *Workflow {
	tb.Helper()
	workflow, err := builder.Build()
	if err != nil {
		tb.Fatalf("Build() error = %v", err)
	}
	return workflow
}
