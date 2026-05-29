package domain

import (
	"strings"
	"testing"
)

func TestImageConfigMarshalYAML(t *testing.T) {
	out, err := ImageConfig{Name: "hashicorp/terraform:1.6"}.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	got, ok := out.(string)
	if !ok {
		t.Fatalf("MarshalYAML() type = %T, want string", out)
	}
	if got != "hashicorp/terraform:1.6" {
		t.Fatalf("MarshalYAML() = %q", got)
	}
}

func TestPipelineToYAMLSortsJobs(t *testing.T) {
	zJob, err := NewJob(JobOptions{Stage: "deploy-0", Script: []string{"echo z"}})
	if err != nil {
		t.Fatal(err)
	}
	aJob, err := NewJob(JobOptions{Stage: "deploy-0", Script: []string{"echo a"}})
	if err != nil {
		t.Fatal(err)
	}
	pipeline, err := NewPipeline(PipelineOptions{
		Stages: []string{"deploy-0"},
		Jobs: []NamedJob{
			{Name: "z-job", Job: zJob},
			{Name: "a-job", Job: aJob},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := pipeline.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error = %v", err)
	}

	yaml := string(out)
	if strings.Index(yaml, "a-job:") > strings.Index(yaml, "z-job:") {
		t.Fatalf("jobs are not sorted in YAML output:\n%s", yaml)
	}
}

func TestNewJobValidatesRequiredFields(t *testing.T) {
	if _, err := NewJob(JobOptions{Script: []string{"echo test"}}); err == nil {
		t.Fatal("NewJob() error = nil, want missing stage error")
	}
	if _, err := NewJob(JobOptions{Stage: "deploy"}); err == nil {
		t.Fatal("NewJob() error = nil, want missing script error")
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
	pipeline, err := NewPipeline(PipelineOptions{
		Stages:    []string{"deploy-0"},
		Variables: map[string]string{"GLOBAL": "true"},
		Jobs:      []NamedJob{{Name: "plan-vpc", Job: job}},
	})
	if err != nil {
		t.Fatal(err)
	}

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
