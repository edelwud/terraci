package generate

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/plugins/gitlab/internal/domain"
)

func TestPipeline_ToYAML(t *testing.T) {
	job, err := domain.NewJob(domain.JobOptions{Stage: "plan-0", Script: []string{"terraform plan"}})
	if err != nil {
		t.Fatal(err)
	}
	builder := domain.NewPipelineBuilder(domain.PipelineOptions{
		Stages:    []string{"plan-0", "apply-0"},
		Variables: map[string]string{"TF_IN_AUTOMATION": "true"},
		Default: &domain.DefaultConfig{
			Image: &domain.ImageConfig{Name: "hashicorp/terraform:1.6"},
		},
	})
	addErr := builder.AddJob("plan-test", job)
	if addErr != nil {
		t.Fatalf("AddJob() error = %v", addErr)
	}
	p, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
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
	if !strings.Contains(yaml, "TF_IN_AUTOMATION") {
		t.Error("YAML should contain explicit user variable")
	}
	if !strings.Contains(yaml, "plan-test:") {
		t.Error("YAML should contain plan-test job")
	}
}
