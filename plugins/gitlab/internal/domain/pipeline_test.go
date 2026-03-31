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
	pipeline := &Pipeline{
		Stages: []string{"deploy-plan-0"},
		Jobs: map[string]*Job{
			"z-job": {Stage: "deploy-plan-0", Script: []string{"echo z"}},
			"a-job": {Stage: "deploy-plan-0", Script: []string{"echo a"}},
		},
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
