package generate

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestGenerator_jobName(t *testing.T) {
	tests := []struct {
		module   *discovery.Module
		jobKind  pipeline.JobKind
		expected string
	}{
		{
			module:   discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
			jobKind:  pipeline.JobKindPlan,
			expected: "plan-platform-stage-eu-central-1-vpc",
		},
		{
			module:   discovery.TestModule("platform", "prod", "us-west-2", "eks"),
			jobKind:  pipeline.JobKindApply,
			expected: "apply-platform-prod-us-west-2-eks",
		},
	}

	for _, tt := range tests {
		result := pipeline.JobName(tt.jobKind, tt.module)
		if result != tt.expected {
			t.Errorf("jobName(%s, %v) = %s, expected %s", tt.module.ID(), tt.jobKind, result, tt.expected)
		}
	}
}

func TestPipeline_ToYAML(t *testing.T) {
	p := &Pipeline{
		Stages:    []string{"plan-0", "apply-0"},
		Variables: map[string]string{"TERRAFORM_BINARY": "terraform"},
		Default: &DefaultConfig{
			Image: &ImageConfig{Name: "hashicorp/terraform:1.6"},
		},
		Jobs: map[string]*Job{
			"plan-test": {
				Stage:  "plan-0",
				Script: []string{"terraform plan"},
			},
		},
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
