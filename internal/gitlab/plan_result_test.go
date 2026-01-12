package gitlab

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

// Sample plan JSON for testing
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
          "bucket": "my-data-bucket"
        },
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

	// Create module directories with plan.json files
	modules := []struct {
		path    string
		content string
	}{
		{
			path:    "platform/stage/eu-central-1/vpc",
			content: samplePlanJSONWithChanges,
		},
		{
			path:    "platform/prod/eu-central-1/eks",
			content: samplePlanJSONNoChanges,
		},
	}

	for _, m := range modules {
		dir := filepath.Join(tmpDir, m.path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
		planFile := filepath.Join(dir, "plan.json")
		if err := os.WriteFile(planFile, []byte(m.content), 0o644); err != nil {
			t.Fatalf("failed to write plan.json: %v", err)
		}
	}

	// Scan results
	collection, err := ScanPlanResults(tmpDir)
	if err != nil {
		t.Fatalf("ScanPlanResults failed: %v", err)
	}

	if len(collection.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(collection.Results))
	}

	// Verify statuses
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

	// Create submodule directory with plan.json
	submodulePath := "platform/stage/eu-central-1/ec2/web"
	dir := filepath.Join(tmpDir, submodulePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	planFile := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planFile, []byte(samplePlanJSONWithChanges), 0o644); err != nil {
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
			service, env, region, module, submodule := parseModulePath(tt.path)

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
		{
			name:     "no changes",
			plan:     &plan.ParsedPlan{},
			expected: "No changes",
		},
		{
			name:     "only adds",
			plan:     &plan.ParsedPlan{ToAdd: 2},
			expected: "+2",
		},
		{
			name:     "only changes",
			plan:     &plan.ParsedPlan{ToChange: 3},
			expected: "~3",
		},
		{
			name:     "only destroys",
			plan:     &plan.ParsedPlan{ToDestroy: 1},
			expected: "-1",
		},
		{
			name:     "mixed changes",
			plan:     &plan.ParsedPlan{ToAdd: 1, ToChange: 2, ToDestroy: 3},
			expected: "+1 ~2 -3",
		},
		{
			name:     "with imports",
			plan:     &plan.ParsedPlan{ToAdd: 1, ToImport: 2},
			expected: "+1 ↓2",
		},
		{
			name:     "all types",
			plan:     &plan.ParsedPlan{ToAdd: 1, ToChange: 2, ToDestroy: 3, ToImport: 4},
			expected: "+1 ~2 -3 ↓4",
		},
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
		{
			name:     "no changes returns empty",
			plan:     &plan.ParsedPlan{},
			contains: []string{},
		},
		{
			name: "create resources",
			plan: &plan.ParsedPlan{
				ToAdd: 2,
				Resources: []plan.ResourceChange{
					{Action: "create", Type: "aws_instance", Name: "web"},
					{Action: "create", Type: "aws_s3_bucket", Name: "data"},
				},
			},
			contains: []string{"**Create:**", "- `aws_instance.web`", "- `aws_s3_bucket.data`"},
		},
		{
			name: "update resources",
			plan: &plan.ParsedPlan{
				ToChange: 1,
				Resources: []plan.ResourceChange{
					{Action: "update", Type: "aws_instance", Name: "web"},
				},
			},
			contains: []string{"**Update:**", "- `aws_instance.web`"},
		},
		{
			name: "delete resources",
			plan: &plan.ParsedPlan{
				ToDestroy: 1,
				Resources: []plan.ResourceChange{
					{Action: "delete", Type: "aws_instance", Name: "old"},
				},
			},
			contains: []string{"**Delete:**", "- `aws_instance.old`"},
		},
		{
			name: "mixed actions grouped",
			plan: &plan.ParsedPlan{
				ToAdd:     1,
				ToChange:  1,
				ToDestroy: 1,
				Resources: []plan.ResourceChange{
					{Action: "create", Type: "aws_instance", Name: "new"},
					{Action: "update", Type: "aws_security_group", Name: "web"},
					{Action: "delete", Type: "aws_s3_bucket", Name: "old"},
				},
			},
			contains: []string{"**Create:**", "- `aws_instance.new`", "**Update:**", "- `aws_security_group.web`", "**Delete:**", "- `aws_s3_bucket.old`"},
		},
		{
			name: "module address preserved",
			plan: &plan.ParsedPlan{
				ToAdd: 1,
				Resources: []plan.ResourceChange{
					{Action: "create", Address: "module.vpc.aws_vpc.main", ModuleAddr: "module.vpc", Type: "aws_vpc", Name: "main"},
				},
			},
			contains: []string{"- `module.vpc.aws_vpc.main`"},
		},
		{
			name: "replace action",
			plan: &plan.ParsedPlan{
				ToAdd:     1,
				ToDestroy: 1,
				Resources: []plan.ResourceChange{
					{Action: "replace", Type: "aws_instance", Name: "web"},
				},
			},
			contains: []string{"**Replace:**", "- `aws_instance.web`"},
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
			name: "filters refreshing state",
			input: `Refreshing state...
data.aws_caller_identity.current: Reading...
data.aws_caller_identity.current: Read complete after 0s

# aws_instance.web will be updated
  ~ resource "aws_instance" "web" {
      ~ instance_type = "t2.micro" -> "t2.small"
    }

Plan: 0 to add, 1 to change, 0 to destroy.`,
			contains: []string{"# aws_instance.web will be updated", "instance_type", "Plan: 0 to add"},
			excludes: []string{"Refreshing state", "Reading..."},
		},
		{
			name: "keeps plan summary",
			input: `Terraform will perform the following actions:

  # aws_instance.web will be created
  + resource "aws_instance" "web" {
      + ami = "ami-12345"
    }

Plan: 1 to add, 0 to change, 0 to destroy.`,
			contains: []string{"Terraform will perform", "aws_instance.web", "Plan: 1 to add"},
		},
		{
			name: "returns original if no diff found",
			input: `Error: Failed to load state

Some error message here`,
			contains: []string{"Error:", "Failed to load state"},
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
					t.Errorf("expected result to NOT contain %q, got:\n%s", s, result)
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
		{
			name:     "no changes",
			plan:     &plan.ParsedPlan{},
			expected: PlanStatusNoChanges,
		},
		{
			name:     "has adds",
			plan:     &plan.ParsedPlan{ToAdd: 1},
			expected: PlanStatusChanges,
		},
		{
			name:     "has changes",
			plan:     &plan.ParsedPlan{ToChange: 1},
			expected: PlanStatusChanges,
		},
		{
			name:     "has destroys",
			plan:     &plan.ParsedPlan{ToDestroy: 1},
			expected: PlanStatusChanges,
		},
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
		{
			name:     "no changes returns 0",
			plan:     &plan.ParsedPlan{},
			expected: 0,
		},
		{
			name:     "has changes returns 2",
			plan:     &plan.ParsedPlan{ToAdd: 1},
			expected: 2,
		},
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
