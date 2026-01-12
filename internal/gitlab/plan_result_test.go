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
		contains []string
	}{
		{
			name: "no changes",
			plan: &plan.ParsedPlan{
				ToAdd:     0,
				ToChange:  0,
				ToDestroy: 0,
			},
			contains: []string{"No changes"},
		},
		{
			name: "only adds",
			plan: &plan.ParsedPlan{
				ToAdd: 1,
				Resources: []plan.ResourceChange{
					{Action: "create", Type: "aws_instance", Name: "web"},
				},
			},
			contains: []string{"+1", "+ aws_instance.web"},
		},
		{
			name: "only changes with attributes",
			plan: &plan.ParsedPlan{
				ToChange: 1,
				Resources: []plan.ResourceChange{
					{
						Action: "update",
						Type:   "aws_instance",
						Name:   "web",
						Attributes: []plan.AttrDiff{
							{Path: "instance_type", OldValue: "t2.micro", NewValue: "t2.small"},
						},
					},
				},
			},
			contains: []string{"~1", "~ aws_instance.web", "instance_type=t2.micro → t2.small"},
		},
		{
			name: "only destroys",
			plan: &plan.ParsedPlan{
				ToDestroy: 1,
				Resources: []plan.ResourceChange{
					{Action: "delete", Type: "aws_instance", Name: "old"},
				},
			},
			contains: []string{"-1", "- aws_instance.old"},
		},
		{
			name: "mixed changes",
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
			contains: []string{"+1 ~1 -1", "+ aws_instance.new", "~ aws_security_group.web", "- aws_s3_bucket.old"},
		},
		{
			name: "with imports",
			plan: &plan.ParsedPlan{
				ToAdd:    1,
				ToImport: 2,
				Resources: []plan.ResourceChange{
					{Action: "create", Type: "aws_instance", Name: "new"},
				},
			},
			contains: []string{"+1 ↓2", "+ aws_instance.new"},
		},
		{
			name: "replace shows symbol",
			plan: &plan.ParsedPlan{
				ToAdd:     1,
				ToDestroy: 1,
				Resources: []plan.ResourceChange{
					{
						Action: "replace",
						Type:   "aws_instance",
						Name:   "web",
						Attributes: []plan.AttrDiff{
							{Path: "ami", OldValue: "ami-old", NewValue: "ami-new", ForceNew: true},
						},
					},
				},
			},
			contains: []string{"+1 -1", "± aws_instance.web", "ami=ami-old → ami-new (forces replacement)"},
		},
		{
			name: "update with multiple attributes",
			plan: &plan.ParsedPlan{
				ToChange: 1,
				Resources: []plan.ResourceChange{
					{
						Action: "update",
						Type:   "aws_instance",
						Name:   "web",
						Attributes: []plan.AttrDiff{
							{Path: "instance_type", OldValue: "t2.micro", NewValue: "t2.small"},
							{Path: "tags.Name", OldValue: "old-name", NewValue: "new-name"},
						},
					},
				},
			},
			contains: []string{"~1", "~ aws_instance.web", "instance_type=t2.micro → t2.small", "tags.Name=old-name → new-name"},
		},
		{
			name: "sensitive attribute",
			plan: &plan.ParsedPlan{
				ToChange: 1,
				Resources: []plan.ResourceChange{
					{
						Action: "update",
						Type:   "aws_db_instance",
						Name:   "main",
						Attributes: []plan.AttrDiff{
							{Path: "password", Sensitive: true},
						},
					},
				},
			},
			contains: []string{"~1", "~ aws_db_instance.main", "password=(sensitive)"},
		},
		{
			name: "computed attribute",
			plan: &plan.ParsedPlan{
				ToChange: 1,
				Resources: []plan.ResourceChange{
					{
						Action: "update",
						Type:   "aws_instance",
						Name:   "web",
						Attributes: []plan.AttrDiff{
							{Path: "public_ip", Computed: true},
						},
					},
				},
			},
			contains: []string{"~1", "~ aws_instance.web", "public_ip=(known after apply)"},
		},
		{
			name: "module prefix preserved",
			plan: &plan.ParsedPlan{
				ToAdd: 1,
				Resources: []plan.ResourceChange{
					{Action: "create", Type: "aws_vpc", Name: "main", Address: "module.vpc.aws_vpc.main", ModuleAddr: "module.vpc"},
				},
			},
			contains: []string{"+1", "+ module.vpc.aws_vpc.main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPlanSummary(tt.plan)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got:\n%s", s, result)
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
