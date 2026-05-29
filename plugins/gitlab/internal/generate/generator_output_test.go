package generate

import (
	"strings"
	"testing"
)

func TestPipeline_ToYAML(t *testing.T) {
	job, err := NewJob(JobOptions{Stage: "plan-0", Script: []string{"terraform plan"}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewPipeline(PipelineOptions{
		Stages:    []string{"plan-0", "apply-0"},
		Variables: map[string]string{"TERRAFORM_BINARY": "terraform"},
		Default: &DefaultConfig{
			Image: &ImageConfig{Name: "hashicorp/terraform:1.6"},
		},
		Jobs: []NamedJob{{Name: "plan-test", Job: job}},
	})
	if err != nil {
		t.Fatal(err)
	}

	yamlBytes, err := p.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "stages:") {
		t.Error("YAML should contain stages")
	}
	if !strings.Contains(yaml, "plan-0") {
		t.Error("YAML should contain plan-0 stage")
	}
	if !strings.Contains(yaml, "TERRAFORM_BINARY") {
		t.Error("YAML should contain TERRAFORM_BINARY variable")
	}
	if !strings.Contains(yaml, "plan-test:") {
		t.Error("YAML should contain plan-test job")
	}
}
