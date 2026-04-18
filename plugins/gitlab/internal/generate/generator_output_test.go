package generate

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestGenerator_jobName(t *testing.T) {
	tests := []struct {
		module   *discovery.Module
		jobType  string
		expected string
	}{
		{
			module:   discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
			jobType:  "plan",
			expected: "plan-platform-stage-eu-central-1-vpc",
		},
		{
			module:   discovery.TestModule("platform", "prod", "us-west-2", "eks"),
			jobType:  "apply",
			expected: "apply-platform-prod-us-west-2-eks",
		},
	}

	for _, tt := range tests {
		result := pipeline.JobName(tt.jobType, tt.module)
		if result != tt.expected {
			t.Errorf("jobName(%s, %s) = %s, expected %s", tt.module.ID(), tt.jobType, result, tt.expected)
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

func TestGenerator_isMREnabled(t *testing.T) {
	tests := []struct {
		name     string
		glCfg    *Config
		expected bool
	}{
		{
			name:     "nil MR config",
			glCfg:    &Config{},
			expected: false,
		},
		{
			name:     "MR config present, no comment config",
			glCfg:    &Config{MR: &MRConfig{}},
			expected: true,
		},
		{
			name:     "MR config present, comment enabled nil",
			glCfg:    &Config{MR: &MRConfig{Comment: &MRCommentConfig{}}},
			expected: true,
		},
		{
			name: "MR config present, comment explicitly enabled",
			glCfg: &Config{MR: &MRConfig{Comment: &MRCommentConfig{
				Enabled: boolPtr(true),
			}}},
			expected: true,
		},
		{
			name: "MR config present, comment explicitly disabled",
			glCfg: &Config{MR: &MRConfig{Comment: &MRCommentConfig{
				Enabled: boolPtr(false),
			}}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator(tt.glCfg, execution.Config{
				Binary:      "terraform",
				InitEnabled: true,
				PlanEnabled: true,
				PlanMode:    execution.PlanModeStandard,
				Parallelism: 4,
			}, nil, graph.NewDependencyGraph(), nil)
			result := gen.IsMREnabled()
			if result != tt.expected {
				t.Errorf("isMREnabled() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
