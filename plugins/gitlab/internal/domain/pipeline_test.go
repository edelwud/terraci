package domain

import (
	"strings"
	"testing"
)

func TestPipelineToYAMLSortsJobs(t *testing.T) {
	zJob, err := NewJob(JobOptions{Stage: "deploy-0", Script: []string{"echo z"}})
	if err != nil {
		t.Fatal(err)
	}
	aJob, err := NewJob(JobOptions{Stage: "deploy-0", Script: []string{"echo a"}})
	if err != nil {
		t.Fatal(err)
	}
	builder := NewPipelineBuilder(PipelineOptions{Stages: []string{"deploy-0"}})
	mustAddPipelineJob(t, builder, "z-job", zJob)
	mustAddPipelineJob(t, builder, "a-job", aJob)
	pipeline := mustBuildPipeline(t, builder)

	out, err := pipeline.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error = %v", err)
	}

	yaml := string(out)
	if strings.Index(yaml, "a-job:") > strings.Index(yaml, "z-job:") {
		t.Fatalf("jobs are not sorted in YAML output:\n%s", yaml)
	}
}

func TestPipelineGettersReturnDefensiveCopies(t *testing.T) {
	tags := []string{"runner"}
	variables := map[string]string{"TF_MODULE": "vpc"}
	needs := []JobNeed{{Job: "plan-vpc"}}
	job, err := NewJob(JobOptions{
		Stage:     "deploy-0",
		Script:    []string{"terraform plan"},
		Tags:      tags,
		Variables: variables,
		Needs:     needs,
	})
	if err != nil {
		t.Fatal(err)
	}
	builder := NewPipelineBuilder(PipelineOptions{
		Stages:    []string{"deploy-0"},
		Variables: map[string]string{"GLOBAL": "true"},
	})
	mustAddPipelineJob(t, builder, "plan-vpc", job)
	pipeline := mustBuildPipeline(t, builder)

	tags[0] = "mutated"
	variables["TF_MODULE"] = "mutated"
	needs[0].Job = "mutated"

	got, ok := pipeline.Job("plan-vpc")
	if !ok {
		t.Fatal("plan-vpc job not found")
	}
	gotTags := got.Tags()
	gotTags[0] = "changed"
	if got.Tags()[0] != "runner" {
		t.Fatalf("Job.Tags() leaked mutation: %#v", got.Tags())
	}
	gotVars := got.Variables()
	gotVars["TF_MODULE"] = "changed"
	if got.Variables()["TF_MODULE"] != "vpc" {
		t.Fatalf("Job.Variables() leaked mutation: %#v", got.Variables())
	}
	gotNeeds := got.Needs()
	gotNeeds[0].Job = "changed"
	if !got.HasNeed("plan-vpc") {
		t.Fatalf("Job.Needs() leaked mutation: %#v", got.Needs())
	}

	stages := pipeline.Stages()
	stages[0] = "changed"
	if pipeline.Stages()[0] != "deploy-0" {
		t.Fatalf("Pipeline.Stages() leaked mutation: %#v", pipeline.Stages())
	}
	pipelineVars := pipeline.Variables()
	pipelineVars["GLOBAL"] = "changed"
	if pipeline.Variables()["GLOBAL"] != "true" {
		t.Fatalf("Pipeline.Variables() leaked mutation: %#v", pipeline.Variables())
	}
}

func TestPipelineBuilderValidatesDuplicateJobs(t *testing.T) {
	job, err := NewJob(JobOptions{Stage: "deploy", Script: []string{"echo test"}})
	if err != nil {
		t.Fatal(err)
	}
	builder := NewPipelineBuilder(PipelineOptions{})
	if err := builder.AddJob("test", job); err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}
	if err := builder.AddJob("test", job); err == nil {
		t.Fatal("AddJob() error = nil, want duplicate job error")
	}
}

func mustAddPipelineJob(tb testing.TB, builder *PipelineBuilder, name string, job Job) {
	tb.Helper()
	if err := builder.AddJob(name, job); err != nil {
		tb.Fatalf("AddJob(%q) error = %v", name, err)
	}
}

func mustBuildPipeline(tb testing.TB, builder *PipelineBuilder) *Pipeline {
	tb.Helper()
	pipeline, err := builder.Build()
	if err != nil {
		tb.Fatalf("Build() error = %v", err)
	}
	return pipeline
}
