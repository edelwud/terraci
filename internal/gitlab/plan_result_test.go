package gitlab

import (
	"os"
	"path/filepath"
	"testing"
)

// samplePlanJSONWithChanges for wrapper tests
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

func TestFormatPlanSummary_Wrapper(t *testing.T) {
	// FormatPlanSummary is a var wrapping ci.FormatPlanSummary
	if FormatPlanSummary == nil {
		t.Fatal("FormatPlanSummary should not be nil")
	}
}

func TestFormatPlanDetails_Wrapper(t *testing.T) {
	if FormatPlanDetails == nil {
		t.Fatal("FormatPlanDetails should not be nil")
	}
}

func TestFilterPlanOutput_Wrapper(t *testing.T) {
	if FilterPlanOutput == nil {
		t.Fatal("FilterPlanOutput should not be nil")
	}

	// Verify it works as a passthrough
	result := FilterPlanOutput("# resource will be updated\n  ~ field")
	if result == "" {
		t.Error("FilterPlanOutput should return non-empty for diff content")
	}
}

func TestScanPlanResults_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	collection, err := ScanPlanResults(tmpDir)
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

	collection, err := ScanPlanResults(tmpDir)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(collection.Results))
	}

	if collection.Results[0].Status != PlanStatusFailed {
		t.Errorf("expected failed status for invalid JSON, got %s", collection.Results[0].Status)
	}
}

// TestScanPlanResults_Wrapper verifies the gitlab package wrapper delegates correctly
func TestScanPlanResults_Wrapper(t *testing.T) {
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "platform", "stage", "eu-central-1", "vpc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte(samplePlanJSONWithChanges), 0o644); err != nil {
		t.Fatalf("failed to write plan.json: %v", err)
	}

	collection, err := ScanPlanResults(tmpDir)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(collection.Results))
	}

	if collection.Results[0].Status != PlanStatusChanges {
		t.Errorf("expected changes status, got %s", collection.Results[0].Status)
	}

	// Verify ToModulePlans still works through wrapper
	plans := collection.ToModulePlans()
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
}
