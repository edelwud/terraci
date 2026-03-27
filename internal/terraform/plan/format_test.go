package plan

import (
	"strings"
	"testing"
)

func TestSummary(t *testing.T) {
	tests := []struct {
		name     string
		plan     *ParsedPlan
		expected string
	}{
		{"no changes", &ParsedPlan{}, "No changes"},
		{"only adds", &ParsedPlan{ToAdd: 2}, "+2"},
		{"only changes", &ParsedPlan{ToChange: 3}, "~3"},
		{"only destroys", &ParsedPlan{ToDestroy: 1}, "-1"},
		{"mixed changes", &ParsedPlan{ToAdd: 1, ToChange: 2, ToDestroy: 3}, "+1 ~2 -3"},
		{"with imports", &ParsedPlan{ToAdd: 1, ToImport: 2}, "+1 ↓2"},
		{"all types", &ParsedPlan{ToAdd: 1, ToChange: 2, ToDestroy: 3, ToImport: 4}, "+1 ~2 -3 ↓4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plan.Summary()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDetails(t *testing.T) {
	tests := []struct {
		name     string
		plan     *ParsedPlan
		contains []string
	}{
		{"no changes returns empty", &ParsedPlan{}, []string{}},
		{
			"create resources",
			&ParsedPlan{
				ToAdd: 2,
				Resources: []ResourceChange{
					{Action: "create", Type: "aws_instance", Name: "web"},
					{Action: "create", Type: "aws_s3_bucket", Name: "data"},
				},
			},
			[]string{"**Create:**", "- `aws_instance.web`", "- `aws_s3_bucket.data`"},
		},
		{
			"mixed actions grouped",
			&ParsedPlan{
				ToAdd: 1, ToChange: 1, ToDestroy: 1,
				Resources: []ResourceChange{
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
			result := tt.plan.Details()
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

func TestFormatResourceAddress(t *testing.T) {
	tests := []struct {
		name string
		rc   ResourceChange
		want string
	}{
		{
			name: "simple address",
			rc:   ResourceChange{Address: "aws_instance.web", Type: "aws_instance", Name: "web"},
			want: "aws_instance.web",
		},
		{
			name: "with module address",
			rc:   ResourceChange{Address: "module.vpc.aws_vpc.main", ModuleAddr: "module.vpc", Type: "aws_vpc", Name: "main"},
			want: "module.vpc.aws_vpc.main",
		},
		{
			name: "empty address falls back to type.name",
			rc:   ResourceChange{Address: "", Type: "aws_s3_bucket", Name: "data"},
			want: "aws_s3_bucket.data",
		},
		{
			name: "very long address gets truncated",
			rc: ResourceChange{
				Address:    "module.very_long_module_name.module.another_nested_module.aws_very_long_resource_type_name.very_long_resource_name_that_exceeds",
				ModuleAddr: "module.very_long_module_name.module.another_nested_module",
				Type:       "aws_very_long_resource_type_name",
				Name:       "very_long_resource_name_that_exceeds",
			},
			want: "module.very_long_module_name.module.another_nested_module.aws_very_long_resou...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatResourceAddress(tt.rc)
			if got != tt.want {
				t.Errorf("formatResourceAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name     string
		plan     *ParsedPlan
		expected int
	}{
		{"no changes returns 0", &ParsedPlan{}, 0},
		{"has changes returns 2", &ParsedPlan{ToAdd: 1}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plan.ExitCode()
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
