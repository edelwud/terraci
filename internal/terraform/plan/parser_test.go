package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseJSONData(t *testing.T) {
	t.Parallel()

	parsed, err := ParseJSONData([]byte(samplePlanJSON))
	if err != nil {
		t.Fatalf("ParseJSONData: %v", err)
	}

	if parsed.TerraformVersion != "1.6.0" {
		t.Errorf("TerraformVersion = %q, want 1.6.0", parsed.TerraformVersion)
	}
	if parsed.ToAdd != 1 {
		t.Errorf("ToAdd = %d, want 1", parsed.ToAdd)
	}
	if parsed.ToChange != 1 {
		t.Errorf("ToChange = %d, want 1", parsed.ToChange)
	}
	if parsed.ToDestroy != 1 {
		t.Errorf("ToDestroy = %d, want 1", parsed.ToDestroy)
	}
	if len(parsed.Resources) != 3 {
		t.Errorf("Resources = %d, want 3", len(parsed.Resources))
	}
}

func TestParseJSONData_NoChanges(t *testing.T) {
	t.Parallel()

	parsed, err := ParseJSONData([]byte(samplePlanJSONNoChanges))
	if err != nil {
		t.Fatalf("ParseJSONData: %v", err)
	}

	if parsed.HasChanges() {
		t.Error("HasChanges() should be false")
	}
	// no-op resources are included (for cost estimation of unchanged infra)
	for _, rc := range parsed.Resources {
		if rc.Action != "no-op" {
			t.Errorf("unexpected action %q for no-change plan", rc.Action)
		}
	}
}

func TestParseJSONData_Replace(t *testing.T) {
	t.Parallel()

	parsed, err := ParseJSONData([]byte(samplePlanJSONReplace))
	if err != nil {
		t.Fatalf("ParseJSONData: %v", err)
	}

	if parsed.ToAdd != 1 || parsed.ToDestroy != 1 {
		t.Errorf("Replace: +%d -%d, want +1 -1", parsed.ToAdd, parsed.ToDestroy)
	}
	if parsed.Resources[0].Action != "replace" {
		t.Errorf("Action = %q, want replace", parsed.Resources[0].Action)
	}
}

func TestParseJSONData_WithModules(t *testing.T) {
	t.Parallel()

	parsed, err := ParseJSONData([]byte(samplePlanJSONWithModule))
	if err != nil {
		t.Fatalf("ParseJSONData: %v", err)
	}

	if len(parsed.Resources) != 2 {
		t.Fatalf("Resources = %d, want 2", len(parsed.Resources))
	}
	if parsed.Resources[0].ModuleAddr != "module.vpc" {
		t.Errorf("[0].ModuleAddr = %q, want module.vpc", parsed.Resources[0].ModuleAddr)
	}
	if parsed.Resources[1].ModuleAddr != "module.vpc.module.subnets" {
		t.Errorf("[1].ModuleAddr = %q", parsed.Resources[1].ModuleAddr)
	}
}

func TestParseJSONData_Sensitive(t *testing.T) {
	t.Parallel()

	parsed, err := ParseJSONData([]byte(samplePlanJSONSensitive))
	if err != nil {
		t.Fatalf("ParseJSONData: %v", err)
	}

	attr := findAttr(t, parsed.Resources[0].Attributes, "password")
	if !attr.Sensitive {
		t.Error("password should be sensitive")
	}
	if attr.OldValue != "(sensitive)" {
		t.Errorf("OldValue = %q, want (sensitive)", attr.OldValue)
	}
	if attr.NewValue != "(sensitive)" {
		t.Errorf("NewValue = %q, want (sensitive)", attr.NewValue)
	}
}

func TestParseJSONData_AttributeDiffs(t *testing.T) {
	t.Parallel()

	parsed, err := ParseJSONData([]byte(samplePlanJSON))
	if err != nil {
		t.Fatalf("ParseJSONData: %v", err)
	}

	update := findResource(t, parsed.Resources, "update")
	attr := findAttr(t, update.Attributes, "instance_type")

	if attr.OldValue != "t2.micro" {
		t.Errorf("OldValue = %q, want t2.micro", attr.OldValue)
	}
	if attr.NewValue != "t2.small" {
		t.Errorf("NewValue = %q, want t2.small", attr.NewValue)
	}
}

func TestParseJSONData_CreateResource(t *testing.T) {
	t.Parallel()

	parsed, err := ParseJSONData([]byte(samplePlanJSON))
	if err != nil {
		t.Fatalf("ParseJSONData: %v", err)
	}

	create := findResource(t, parsed.Resources, "create")
	if create.Type != "aws_s3_bucket" {
		t.Errorf("Type = %q, want aws_s3_bucket", create.Type)
	}

	attr := findAttr(t, create.Attributes, "bucket")
	if attr.NewValue != "my-data-bucket" {
		t.Errorf("bucket NewValue = %q, want my-data-bucket", attr.NewValue)
	}
}

func TestParseJSON_File(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(path, []byte(samplePlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseJSON(path)
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if parsed.ToAdd != 1 || parsed.ToChange != 1 || parsed.ToDestroy != 1 {
		t.Errorf("+%d ~%d -%d, want +1 ~1 -1", parsed.ToAdd, parsed.ToChange, parsed.ToDestroy)
	}
}

func TestParseJSON_InvalidJSON(t *testing.T) {
	t.Parallel()

	if _, err := ParseJSONData([]byte("not json")); err == nil {
		t.Error("expected error")
	}
}

func TestParseJSON_FileNotFound(t *testing.T) {
	t.Parallel()

	if _, err := ParseJSON("/nonexistent/plan.json"); err == nil {
		t.Error("expected error")
	}
}

// --- helpers ---

func findResource(t *testing.T, resources []ResourceChange, action string) ResourceChange {
	t.Helper()
	for _, r := range resources {
		if r.Action == action {
			return r
		}
	}
	t.Fatalf("resource with action %q not found", action)
	return ResourceChange{}
}

func findAttr(t *testing.T, attrs []AttrDiff, path string) AttrDiff {
	t.Helper()
	for _, a := range attrs {
		if a.Path == path {
			return a
		}
	}
	t.Fatalf("attribute %q not found", path)
	return AttrDiff{}
}
