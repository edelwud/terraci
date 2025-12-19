package gitlab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
				Summary:     "Plan: 1 to add",
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

func TestParsePlanOutput(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		exitCode       int
		expectedStatus PlanStatus
		expectedHas    string
	}{
		{
			name:           "no changes exit 0",
			output:         "No changes. Your infrastructure matches the configuration.",
			exitCode:       0,
			expectedStatus: PlanStatusNoChanges,
			expectedHas:    "No changes",
		},
		{
			name:           "has changes exit 2",
			output:         "Plan: 2 to add, 1 to change, 0 to destroy.",
			exitCode:       2,
			expectedStatus: PlanStatusChanges,
			expectedHas:    "Plan: 2 to add",
		},
		{
			name:           "error exit 1",
			output:         "Error: some error",
			exitCode:       1,
			expectedStatus: PlanStatusFailed,
			expectedHas:    "",
		},
		{
			name: "multiline output with plan",
			output: `Refreshing state...
data.aws_vpc.main: Reading...
data.aws_vpc.main: Read complete

Plan: 3 to add, 0 to change, 1 to destroy.`,
			exitCode:       2,
			expectedStatus: PlanStatusChanges,
			expectedHas:    "Plan: 3 to add",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, summary := ParsePlanOutput(tt.output, tt.exitCode)

			if status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, status)
			}

			if tt.expectedHas != "" && summary == "" {
				t.Error("expected non-empty summary")
			}

			if tt.expectedHas != "" && summary != "" {
				if !strings.Contains(summary, tt.expectedHas) {
					t.Errorf("expected summary to contain %q, got %q", tt.expectedHas, summary)
				}
			}
		})
	}
}

func TestScanPlanResults(t *testing.T) {
	// Create temp directory with test structure
	tmpDir := t.TempDir()

	// Create module directories with plan.txt files
	modules := []struct {
		path    string
		content string
	}{
		{
			path:    "platform/stage/eu-central-1/vpc",
			content: "Plan: 1 to add, 0 to change, 0 to destroy.",
		},
		{
			path:    "platform/prod/eu-central-1/eks",
			content: "No changes. Your infrastructure matches the configuration.",
		},
	}

	for _, m := range modules {
		dir := filepath.Join(tmpDir, m.path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
		planFile := filepath.Join(dir, "plan.txt")
		if err := os.WriteFile(planFile, []byte(m.content), 0o644); err != nil {
			t.Fatalf("failed to write plan.txt: %v", err)
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

func TestInferExitCode(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected int
	}{
		{
			name:     "no changes",
			output:   "No changes. Your infrastructure matches the configuration.",
			expected: 0,
		},
		{
			name:     "has changes",
			output:   "Plan: 1 to add, 0 to change, 0 to destroy.",
			expected: 2,
		},
		{
			name:     "error",
			output:   "Error: some terraform error",
			expected: 1,
		},
		{
			name:     "unknown",
			output:   "some random output",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferExitCode(tt.output)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
