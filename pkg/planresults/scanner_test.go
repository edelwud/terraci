package planresults

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/pkg/ci"
)

func TestPlanResultCollection_ToModulePlans(t *testing.T) {
	collection := &ci.PlanResultCollection{
		Results: []ci.PlanResult{
			{
				ModuleID:   "platform/stage/eu-central-1/vpc",
				ModulePath: "platform/stage/eu-central-1/vpc",
				Components: map[string]string{
					"service":     "platform",
					"environment": "stage",
					"region":      "eu-central-1",
					"module":      "vpc",
				},
				Status:  ci.PlanStatusChanges,
				Summary: "+1 (aws_vpc)",
			},
		},
	}

	plans := collection.ToModulePlans()

	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	if plans[0].ModuleID != "platform/stage/eu-central-1/vpc" {
		t.Errorf("unexpected module ID: %s", plans[0].ModuleID)
	}

	if plans[0].Status != ci.PlanStatusChanges {
		t.Errorf("expected status %s, got %s", ci.PlanStatusChanges, plans[0].Status)
	}
}

const samplePlanJSONWithChanges = `{
  "format_version": "1.2",
  "terraform_version": "1.6.0",
  "resource_changes": [
    {
      "address": "aws_instance.web",
      "module_address": "",
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["update"],
        "before": {"ami": "ami-12345", "instance_type": "t2.micro"},
        "after": {"ami": "ami-12345", "instance_type": "t2.small"},
        "after_unknown": {},
        "before_sensitive": {},
        "after_sensitive": {}
      }
    },
    {
      "address": "aws_s3_bucket.data",
      "module_address": "",
      "mode": "managed",
      "type": "aws_s3_bucket",
      "name": "data",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {"bucket": "my-data-bucket"},
        "after_unknown": {"id": true},
        "before_sensitive": {},
        "after_sensitive": {}
      }
    }
  ]
}`

const samplePlanJSONNoChanges = `{
  "format_version": "1.2",
  "terraform_version": "1.6.0",
  "resource_changes": [
    {
      "address": "aws_instance.web",
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "change": {
        "actions": ["no-op"],
        "before": {"ami": "ami-12345"},
        "after": {"ami": "ami-12345"}
      }
    }
  ]
}`

func TestScan(t *testing.T) {
	tmpDir := t.TempDir()

	modules := []struct {
		path    string
		content string
	}{
		{"platform/stage/eu-central-1/vpc", samplePlanJSONWithChanges},
		{"platform/prod/eu-central-1/eks", samplePlanJSONNoChanges},
	}

	for _, m := range modules {
		dir := filepath.Join(tmpDir, m.path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte(m.content), 0o644); err != nil {
			t.Fatalf("failed to write plan.json: %v", err)
		}
	}

	collection, err := Scan(tmpDir, nil)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(collection.Results))
	}

	statusMap := make(map[string]ci.PlanStatus)
	for i := range collection.Results {
		statusMap[collection.Results[i].ModuleID] = collection.Results[i].Status
	}

	if statusMap["platform/stage/eu-central-1/vpc"] != ci.PlanStatusChanges {
		t.Errorf("vpc should have changes status, got %s", statusMap["platform/stage/eu-central-1/vpc"])
	}

	if statusMap["platform/prod/eu-central-1/eks"] != ci.PlanStatusNoChanges {
		t.Errorf("eks should have no_changes status, got %s", statusMap["platform/prod/eu-central-1/eks"])
	}
}

func TestScanPlanResults_WithSubmodule(t *testing.T) {
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "platform", "stage", "eu-central-1", "ec2", "web")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte(samplePlanJSONWithChanges), 0o644); err != nil {
		t.Fatalf("failed to write plan.json: %v", err)
	}

	collection, err := Scan(tmpDir, nil)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(collection.Results))
	}

	result := collection.Results[0]
	if result.Get("module") != "ec2" {
		t.Errorf("expected module 'ec2', got %q", result.Get("module"))
	}
	if result.Get("submodule") != "web" {
		t.Errorf("expected submodule 'web', got %q", result.Get("submodule"))
	}
}

func TestScanPlanResults_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	collection, err := Scan(tmpDir, nil)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(collection.Results))
	}
}

func TestGetPlanStatus(t *testing.T) {
	tests := []struct {
		name     string
		plan     *plan.ParsedPlan
		expected ci.PlanStatus
	}{
		{"no changes", &plan.ParsedPlan{}, ci.PlanStatusNoChanges},
		{"has adds", &plan.ParsedPlan{ToAdd: 1}, ci.PlanStatusChanges},
		{"has changes", &plan.ParsedPlan{ToChange: 1}, ci.PlanStatusChanges},
		{"has destroys", &plan.ParsedPlan{ToDestroy: 1}, ci.PlanStatusChanges},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ci.PlanStatusFromPlan(tt.plan.HasChanges())
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
