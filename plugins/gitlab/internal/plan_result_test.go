package gitlabci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

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
    }
  ]
}`

func TestScanPlanResults_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	collection, err := ci.ScanPlanResults(tmpDir, nil)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}
	if len(collection.Results) != 0 {
		t.Errorf("expected 0 results for empty dir, got %d", len(collection.Results))
	}
}

func TestScanPlanResults_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "platform", "stage", "eu-central-1", "vpc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to write plan.json: %v", err)
	}

	collection, err := ci.ScanPlanResults(tmpDir, nil)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(collection.Results))
	}

	if collection.Results[0].Status != ci.PlanStatusFailed {
		t.Errorf("expected failed status for invalid JSON, got %s", collection.Results[0].Status)
	}
}

func TestScanPlanResults_WithChanges(t *testing.T) {
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "platform", "stage", "eu-central-1", "vpc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte(samplePlanJSONWithChanges), 0o644); err != nil {
		t.Fatalf("failed to write plan.json: %v", err)
	}

	collection, err := ci.ScanPlanResults(tmpDir, nil)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(collection.Results))
	}

	if collection.Results[0].Status != ci.PlanStatusChanges {
		t.Errorf("expected changes status, got %s", collection.Results[0].Status)
	}

	plans := collection.ToModulePlans()
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
}
