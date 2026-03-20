package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/internal/terraform/plan"
)

func TestPlanResultCollection_ToModulePlans(t *testing.T) {
	collection := &PlanResultCollection{
		Results: []PlanResult{
			{
				ModuleID:    "platform/stage/eu-central-1/vpc",
				ModulePath:  "platform/stage/eu-central-1/vpc",
				Service:     "platform",
				Environment: "stage",
				Region:      "eu-central-1",
				Module:      "vpc",
				Status:      PlanStatusChanges,
				Summary:     "+1 (aws_vpc)",
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

	if plans[0].Status != PlanStatusChanges {
		t.Errorf("expected status %s, got %s", PlanStatusChanges, plans[0].Status)
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

func TestScanPlanResults(t *testing.T) {
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

	collection, err := ScanPlanResults(tmpDir)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(collection.Results))
	}

	statusMap := make(map[string]PlanStatus)
	for i := range collection.Results {
		statusMap[collection.Results[i].ModuleID] = collection.Results[i].Status
	}

	if statusMap["platform/stage/eu-central-1/vpc"] != PlanStatusChanges {
		t.Errorf("vpc should have changes status, got %s", statusMap["platform/stage/eu-central-1/vpc"])
	}

	if statusMap["platform/prod/eu-central-1/eks"] != PlanStatusNoChanges {
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

	collection, err := ScanPlanResults(tmpDir)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(collection.Results))
	}

	result := collection.Results[0]
	if result.Module != "ec2" {
		t.Errorf("expected module 'ec2', got %q", result.Module)
	}
	if result.Submodule != "web" {
		t.Errorf("expected submodule 'web', got %q", result.Submodule)
	}
}

func TestScanPlanResults_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	collection, err := ScanPlanResults(tmpDir)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(collection.Results))
	}
}

func TestParseModulePath(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		expectedService   string
		expectedEnv       string
		expectedRegion    string
		expectedModule    string
		expectedSubmodule string
	}{
		{
			name:            "regular module",
			path:            "platform/stage/eu-central-1/vpc",
			expectedService: "platform",
			expectedEnv:     "stage",
			expectedRegion:  "eu-central-1",
			expectedModule:  "vpc",
		},
		{
			name:              "submodule",
			path:              "platform/stage/eu-central-1/ec2/web",
			expectedService:   "platform",
			expectedEnv:       "stage",
			expectedRegion:    "eu-central-1",
			expectedModule:    "ec2",
			expectedSubmodule: "web",
		},
		{
			name:              "nested submodule",
			path:              "platform/stage/eu-central-1/ec2/web/api",
			expectedService:   "platform",
			expectedEnv:       "stage",
			expectedRegion:    "eu-central-1",
			expectedModule:    "ec2",
			expectedSubmodule: "web/api",
		},
		{
			name: "short path",
			path: "foo/bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, env, region, module, submodule := ParseModulePath(tt.path)

			if service != tt.expectedService {
				t.Errorf("service: expected %q, got %q", tt.expectedService, service)
			}
			if env != tt.expectedEnv {
				t.Errorf("env: expected %q, got %q", tt.expectedEnv, env)
			}
			if region != tt.expectedRegion {
				t.Errorf("region: expected %q, got %q", tt.expectedRegion, region)
			}
			if module != tt.expectedModule {
				t.Errorf("module: expected %q, got %q", tt.expectedModule, module)
			}
			if submodule != tt.expectedSubmodule {
				t.Errorf("submodule: expected %q, got %q", tt.expectedSubmodule, submodule)
			}
		})
	}
}

func TestFormatPlanSummary(t *testing.T) {
	tests := []struct {
		name     string
		plan     *plan.ParsedPlan
		expected string
	}{
		{"no changes", &plan.ParsedPlan{}, "No changes"},
		{"only adds", &plan.ParsedPlan{ToAdd: 2}, "+2"},
		{"only changes", &plan.ParsedPlan{ToChange: 3}, "~3"},
		{"only destroys", &plan.ParsedPlan{ToDestroy: 1}, "-1"},
		{"mixed changes", &plan.ParsedPlan{ToAdd: 1, ToChange: 2, ToDestroy: 3}, "+1 ~2 -3"},
		{"with imports", &plan.ParsedPlan{ToAdd: 1, ToImport: 2}, "+1 ↓2"},
		{"all types", &plan.ParsedPlan{ToAdd: 1, ToChange: 2, ToDestroy: 3, ToImport: 4}, "+1 ~2 -3 ↓4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPlanSummary(tt.plan)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatPlanDetails(t *testing.T) {
	tests := []struct {
		name     string
		plan     *plan.ParsedPlan
		contains []string
	}{
		{"no changes returns empty", &plan.ParsedPlan{}, []string{}},
		{
			"create resources",
			&plan.ParsedPlan{
				ToAdd: 2,
				Resources: []plan.ResourceChange{
					{Action: "create", Type: "aws_instance", Name: "web"},
					{Action: "create", Type: "aws_s3_bucket", Name: "data"},
				},
			},
			[]string{"**Create:**", "- `aws_instance.web`", "- `aws_s3_bucket.data`"},
		},
		{
			"mixed actions grouped",
			&plan.ParsedPlan{
				ToAdd: 1, ToChange: 1, ToDestroy: 1,
				Resources: []plan.ResourceChange{
					{Action: "create", Type: "aws_instance", Name: "new"},
					{Action: "update", Type: "aws_security_group", Name: "web"},
					{Action: "delete", Type: "aws_s3_bucket", Name: "old"},
				},
			},
			[]string{"**Create:**", "**Update:**", "**Delete:**"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPlanDetails(tt.plan)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got:\n%s", s, result)
				}
			}
		})
	}
}

func TestFilterPlanOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			"filters refreshing state",
			"Refreshing state...\ndata.aws_caller_identity.current: Reading...\n\n# aws_instance.web will be updated\n  ~ instance_type\n\nPlan: 0 to add, 1 to change",
			[]string{"# aws_instance.web will be updated", "Plan: 0 to add"},
			[]string{"Refreshing state", "Reading..."},
		},
		{
			"returns original if no diff",
			"Error: Failed to load state\n\nSome error message",
			[]string{"Error:", "Failed to load state"},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterPlanOutput(tt.input)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got:\n%s", s, result)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(result, s) {
					t.Errorf("expected result to NOT contain %q", s)
				}
			}
		})
	}
}

func TestGetPlanStatus(t *testing.T) {
	tests := []struct {
		name     string
		plan     *plan.ParsedPlan
		expected PlanStatus
	}{
		{"no changes", &plan.ParsedPlan{}, PlanStatusNoChanges},
		{"has adds", &plan.ParsedPlan{ToAdd: 1}, PlanStatusChanges},
		{"has changes", &plan.ParsedPlan{ToChange: 1}, PlanStatusChanges},
		{"has destroys", &plan.ParsedPlan{ToDestroy: 1}, PlanStatusChanges},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPlanStatus(tt.plan)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name     string
		plan     *plan.ParsedPlan
		expected int
	}{
		{"no changes returns 0", &plan.ParsedPlan{}, 0},
		{"has changes returns 2", &plan.ParsedPlan{ToAdd: 1}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getExitCode(tt.plan)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
