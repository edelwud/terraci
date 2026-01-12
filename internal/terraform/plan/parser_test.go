package plan

import (
	"os"
	"path/filepath"
	"testing"
)

// Sample plan JSON for testing
const samplePlanJSON = `{
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
        "before": {
          "ami": "ami-12345",
          "instance_type": "t2.micro",
          "tags": {"Name": "old-name"}
        },
        "after": {
          "ami": "ami-12345",
          "instance_type": "t2.small",
          "tags": {"Name": "new-name"}
        },
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
        "after": {
          "bucket": "my-data-bucket",
          "tags": {}
        },
        "after_unknown": {"id": true},
        "before_sensitive": {},
        "after_sensitive": {}
      }
    },
    {
      "address": "aws_instance.old",
      "module_address": "",
      "mode": "managed",
      "type": "aws_instance",
      "name": "old",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["delete"],
        "before": {
          "ami": "ami-old",
          "instance_type": "t2.micro"
        },
        "after": null,
        "after_unknown": {},
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

const samplePlanJSONReplace = `{
  "format_version": "1.2",
  "terraform_version": "1.6.0",
  "resource_changes": [
    {
      "address": "aws_instance.web",
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "change": {
        "actions": ["delete", "create"],
        "before": {"ami": "ami-old"},
        "after": {"ami": "ami-new"},
        "replace_paths": [["ami"]]
      }
    }
  ]
}`

const samplePlanJSONWithModule = `{
  "format_version": "1.2",
  "terraform_version": "1.6.0",
  "resource_changes": [
    {
      "address": "module.vpc.aws_vpc.main",
      "module_address": "module.vpc",
      "mode": "managed",
      "type": "aws_vpc",
      "name": "main",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {"cidr_block": "10.0.0.0/16"}
      }
    },
    {
      "address": "module.vpc.module.subnets.aws_subnet.private[0]",
      "module_address": "module.vpc.module.subnets",
      "mode": "managed",
      "type": "aws_subnet",
      "name": "private",
      "index": 0,
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {"cidr_block": "10.0.1.0/24"}
      }
    }
  ]
}`

const samplePlanJSONSensitive = `{
  "format_version": "1.2",
  "terraform_version": "1.6.0",
  "resource_changes": [
    {
      "address": "aws_db_instance.main",
      "mode": "managed",
      "type": "aws_db_instance",
      "name": "main",
      "change": {
        "actions": ["update"],
        "before": {"password": "old-secret"},
        "after": {"password": "new-secret"},
        "before_sensitive": {"password": true},
        "after_sensitive": {"password": true}
      }
    }
  ]
}`

func TestParseJSONData(t *testing.T) {
	parsed, err := ParseJSONData([]byte(samplePlanJSON))
	if err != nil {
		t.Fatalf("ParseJSONData failed: %v", err)
	}

	if parsed.TerraformVersion != "1.6.0" {
		t.Errorf("TerraformVersion: expected 1.6.0, got %s", parsed.TerraformVersion)
	}

	if parsed.ToAdd != 1 {
		t.Errorf("ToAdd: expected 1, got %d", parsed.ToAdd)
	}

	if parsed.ToChange != 1 {
		t.Errorf("ToChange: expected 1, got %d", parsed.ToChange)
	}

	if parsed.ToDestroy != 1 {
		t.Errorf("ToDestroy: expected 1, got %d", parsed.ToDestroy)
	}

	if len(parsed.Resources) != 3 {
		t.Errorf("Resources count: expected 3, got %d", len(parsed.Resources))
	}
}

func TestParseJSONData_NoChanges(t *testing.T) {
	parsed, err := ParseJSONData([]byte(samplePlanJSONNoChanges))
	if err != nil {
		t.Fatalf("ParseJSONData failed: %v", err)
	}

	if parsed.ToAdd != 0 || parsed.ToChange != 0 || parsed.ToDestroy != 0 {
		t.Errorf("Expected no changes, got +%d ~%d -%d", parsed.ToAdd, parsed.ToChange, parsed.ToDestroy)
	}

	if parsed.HasChanges() {
		t.Error("HasChanges() should return false")
	}
}

func TestParseJSONData_Replace(t *testing.T) {
	parsed, err := ParseJSONData([]byte(samplePlanJSONReplace))
	if err != nil {
		t.Fatalf("ParseJSONData failed: %v", err)
	}

	// Replace counts as both add and destroy
	if parsed.ToAdd != 1 {
		t.Errorf("ToAdd: expected 1 (from replace), got %d", parsed.ToAdd)
	}

	if parsed.ToDestroy != 1 {
		t.Errorf("ToDestroy: expected 1 (from replace), got %d", parsed.ToDestroy)
	}

	if len(parsed.Resources) != 1 {
		t.Fatalf("Resources count: expected 1, got %d", len(parsed.Resources))
	}

	if parsed.Resources[0].Action != "replace" {
		t.Errorf("Action: expected replace, got %s", parsed.Resources[0].Action)
	}
}

func TestParseJSONData_WithModules(t *testing.T) {
	parsed, err := ParseJSONData([]byte(samplePlanJSONWithModule))
	if err != nil {
		t.Fatalf("ParseJSONData failed: %v", err)
	}

	if len(parsed.Resources) != 2 {
		t.Fatalf("Resources count: expected 2, got %d", len(parsed.Resources))
	}

	// Check first resource (aws_vpc)
	r1 := parsed.Resources[0]
	if r1.Type != "aws_vpc" {
		t.Errorf("Resource[0].Type: expected aws_vpc, got %s", r1.Type)
	}
	if r1.ModuleAddr != "module.vpc" {
		t.Errorf("Resource[0].ModuleAddr: expected module.vpc, got %s", r1.ModuleAddr)
	}

	// Check second resource (aws_subnet)
	r2 := parsed.Resources[1]
	if r2.Type != "aws_subnet" {
		t.Errorf("Resource[1].Type: expected aws_subnet, got %s", r2.Type)
	}
	if r2.ModuleAddr != "module.vpc.module.subnets" {
		t.Errorf("Resource[1].ModuleAddr: expected module.vpc.module.subnets, got %s", r2.ModuleAddr)
	}
}

func TestParseJSONData_SensitiveValues(t *testing.T) {
	parsed, err := ParseJSONData([]byte(samplePlanJSONSensitive))
	if err != nil {
		t.Fatalf("ParseJSONData failed: %v", err)
	}

	if len(parsed.Resources) != 1 {
		t.Fatalf("Resources count: expected 1, got %d", len(parsed.Resources))
	}

	// Find password attribute
	var passwordAttr *AttrDiff
	for i := range parsed.Resources[0].Attributes {
		if parsed.Resources[0].Attributes[i].Path == "password" {
			passwordAttr = &parsed.Resources[0].Attributes[i]
			break
		}
	}

	if passwordAttr == nil {
		t.Fatal("Password attribute not found")
	}

	if !passwordAttr.Sensitive {
		t.Error("Password should be marked as sensitive")
	}

	if passwordAttr.OldValue != "(sensitive)" {
		t.Errorf("Old value should be (sensitive), got %s", passwordAttr.OldValue)
	}

	if passwordAttr.NewValue != "(sensitive)" {
		t.Errorf("New value should be (sensitive), got %s", passwordAttr.NewValue)
	}
}

func TestParseJSONData_AttributeDiffs(t *testing.T) {
	parsed, err := ParseJSONData([]byte(samplePlanJSON))
	if err != nil {
		t.Fatalf("ParseJSONData failed: %v", err)
	}

	// Find the update resource
	var updateResource *ResourceChange
	for i := range parsed.Resources {
		if parsed.Resources[i].Action == "update" {
			updateResource = &parsed.Resources[i]
			break
		}
	}

	if updateResource == nil {
		t.Fatal("Update resource not found")
	}

	// Should have attribute changes
	if len(updateResource.Attributes) == 0 {
		t.Error("Update resource should have attribute changes")
	}

	// Find instance_type attribute
	var instanceTypeAttr *AttrDiff
	for i := range updateResource.Attributes {
		if updateResource.Attributes[i].Path == "instance_type" {
			instanceTypeAttr = &updateResource.Attributes[i]
			break
		}
	}

	if instanceTypeAttr == nil {
		t.Fatal("instance_type attribute not found")
	}

	if instanceTypeAttr.OldValue != "t2.micro" {
		t.Errorf("instance_type OldValue: expected t2.micro, got %s", instanceTypeAttr.OldValue)
	}

	if instanceTypeAttr.NewValue != "t2.small" {
		t.Errorf("instance_type NewValue: expected t2.small, got %s", instanceTypeAttr.NewValue)
	}
}

func TestParseJSON_File(t *testing.T) {
	// Create temp file with plan JSON
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.json")

	if err := os.WriteFile(planPath, []byte(samplePlanJSON), 0o644); err != nil {
		t.Fatalf("Failed to write plan file: %v", err)
	}

	parsed, err := ParseJSON(planPath)
	if err != nil {
		t.Fatalf("ParseJSON failed: %v", err)
	}

	if parsed.ToAdd != 1 || parsed.ToChange != 1 || parsed.ToDestroy != 1 {
		t.Errorf("Unexpected counts: +%d ~%d -%d", parsed.ToAdd, parsed.ToChange, parsed.ToDestroy)
	}
}

func TestParseJSON_InvalidJSON(t *testing.T) {
	_, err := ParseJSONData([]byte("not valid json"))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseJSON_FileNotFound(t *testing.T) {
	_, err := ParseJSON("/nonexistent/path/plan.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestParsedPlan_HasChanges(t *testing.T) {
	tests := []struct {
		name     string
		plan     ParsedPlan
		expected bool
	}{
		{
			name:     "no changes",
			plan:     ParsedPlan{},
			expected: false,
		},
		{
			name:     "has add",
			plan:     ParsedPlan{ToAdd: 1},
			expected: true,
		},
		{
			name:     "has change",
			plan:     ParsedPlan{ToChange: 1},
			expected: true,
		},
		{
			name:     "has destroy",
			plan:     ParsedPlan{ToDestroy: 1},
			expected: true,
		},
		{
			name:     "has all",
			plan:     ParsedPlan{ToAdd: 1, ToChange: 2, ToDestroy: 3},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.HasChanges(); got != tt.expected {
				t.Errorf("HasChanges() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{nil, ""},
		{"hello", "hello"},
		{true, "true"},
		{false, "false"},
		{float64(42), "42"},
		{float64(3.14), "3.14"},
		{[]interface{}{}, "[]"},
		{[]interface{}{1, 2, 3}, "[3 items]"},
		{map[string]interface{}{}, "{}"},
		{map[string]interface{}{"a": 1, "b": 2}, "{2 keys}"},
	}

	for _, tt := range tests {
		result := formatValue(tt.input)
		if result != tt.expected {
			t.Errorf("formatValue(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetNestedValue(t *testing.T) {
	m := map[string]interface{}{
		"a": "value_a",
		"b": map[string]interface{}{
			"c": "value_c",
			"d": map[string]interface{}{
				"e": "value_e",
			},
		},
	}

	tests := []struct {
		key      string
		expected interface{}
	}{
		{"a", "value_a"},
		{"b.c", "value_c"},
		{"b.d.e", "value_e"},
		{"nonexistent", nil},
		{"b.nonexistent", nil},
	}

	for _, tt := range tests {
		result := getNestedValue(m, tt.key)
		if result != tt.expected {
			t.Errorf("getNestedValue(%q) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestResourceChange_AttributeAccess(t *testing.T) {
	parsed, err := ParseJSONData([]byte(samplePlanJSON))
	if err != nil {
		t.Fatalf("ParseJSONData failed: %v", err)
	}

	// Find create resource (aws_s3_bucket)
	var createResource *ResourceChange
	for i := range parsed.Resources {
		if parsed.Resources[i].Action == "create" {
			createResource = &parsed.Resources[i]
			break
		}
	}

	if createResource == nil {
		t.Fatal("Create resource not found")
	}

	if createResource.Type != "aws_s3_bucket" {
		t.Errorf("Type: expected aws_s3_bucket, got %s", createResource.Type)
	}

	if createResource.Name != "data" {
		t.Errorf("Name: expected data, got %s", createResource.Name)
	}

	// Find bucket attribute (new attribute in create action)
	var bucketAttr *AttrDiff
	for i := range createResource.Attributes {
		if createResource.Attributes[i].Path == "bucket" {
			bucketAttr = &createResource.Attributes[i]
			break
		}
	}

	if bucketAttr == nil {
		t.Fatal("bucket attribute not found")
	}

	if bucketAttr.NewValue != "my-data-bucket" {
		t.Errorf("bucket NewValue: expected my-data-bucket, got %s", bucketAttr.NewValue)
	}
}
