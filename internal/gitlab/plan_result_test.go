package gitlab

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSavePlanResult(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	result := &PlanResult{
		ModuleID:    "platform/stage/eu-central-1/vpc",
		ModulePath:  "platform/stage/eu-central-1/vpc",
		Service:     "platform",
		Environment: "stage",
		Region:      "eu-central-1",
		Module:      "vpc",
		Status:      PlanStatusChanges,
		Summary:     "Plan: 1 to add, 0 to change, 0 to destroy.",
		Details:     "terraform plan output here",
		ExitCode:    2,
		Duration:    5 * time.Second,
		StartedAt:   time.Now().UTC(),
		FinishedAt:  time.Now().UTC(),
	}

	err := SavePlanResult(result, tmpDir)
	if err != nil {
		t.Fatalf("SavePlanResult failed: %v", err)
	}

	// Check file exists
	expectedFile := filepath.Join(tmpDir, "plan-platform-stage-eu-central-1-vpc.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", expectedFile)
	}
}

func TestLoadPlanResults(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create two test results
	results := []*PlanResult{
		{
			ModuleID:    "platform/stage/eu-central-1/vpc",
			ModulePath:  "platform/stage/eu-central-1/vpc",
			Service:     "platform",
			Environment: "stage",
			Region:      "eu-central-1",
			Module:      "vpc",
			Status:      PlanStatusChanges,
			Summary:     "Plan: 1 to add, 0 to change, 0 to destroy.",
		},
		{
			ModuleID:    "platform/prod/eu-central-1/eks",
			ModulePath:  "platform/prod/eu-central-1/eks",
			Service:     "platform",
			Environment: "prod",
			Region:      "eu-central-1",
			Module:      "eks",
			Status:      PlanStatusNoChanges,
			Summary:     "No changes. Infrastructure is up-to-date.",
		},
	}

	for _, r := range results {
		if err := SavePlanResult(r, tmpDir); err != nil {
			t.Fatalf("failed to save result: %v", err)
		}
	}

	// Load and verify
	collection, err := LoadPlanResults(tmpDir)
	if err != nil {
		t.Fatalf("LoadPlanResults failed: %v", err)
	}

	if len(collection.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(collection.Results))
	}
}

func TestLoadPlanResults_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	collection, err := LoadPlanResults(tmpDir)
	if err != nil {
		t.Fatalf("LoadPlanResults failed: %v", err)
	}

	if len(collection.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(collection.Results))
	}
}

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
				Duration:    5 * time.Second,
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

	if plans[0].Duration != 5*time.Second {
		t.Errorf("expected duration 5s, got %s", plans[0].Duration)
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
				if !contains(summary, tt.expectedHas) {
					t.Errorf("expected summary to contain %q, got %q", tt.expectedHas, summary)
				}
			}
		})
	}
}

func TestPlanResultWriter(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewPlanResultWriter("platform/stage/eu-central-1/vpc", "platform/stage/eu-central-1/vpc", tmpDir)

	// Check initial state
	result := writer.Result()
	if result.Status != PlanStatusPending {
		t.Errorf("expected initial status pending, got %s", result.Status)
	}

	if result.Service != "platform" {
		t.Errorf("expected service 'platform', got %s", result.Service)
	}

	if result.Environment != "stage" {
		t.Errorf("expected env 'stage', got %s", result.Environment)
	}

	// Set output
	output := "Plan: 1 to add, 0 to change, 0 to destroy."
	writer.SetOutput(output, 2)

	if result.Status != PlanStatusChanges {
		t.Errorf("expected status changes, got %s", result.Status)
	}

	// Finish
	if err := writer.Finish(); err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	// Verify file was created
	expectedFile := filepath.Join(tmpDir, "plan-platform-stage-eu-central-1-vpc.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", expectedFile)
	}

	// Verify duration was set (FinishedAt should be >= StartedAt)
	if result.FinishedAt.Before(result.StartedAt) {
		t.Error("expected FinishedAt to be after StartedAt")
	}
}

func TestPlanResultWriter_SetError(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewPlanResultWriter("platform/stage/eu-central-1/vpc", "platform/stage/eu-central-1/vpc", tmpDir)

	err := &testError{msg: "terraform init failed"}
	writer.SetError(err)

	result := writer.Result()
	if result.Status != PlanStatusFailed {
		t.Errorf("expected status failed, got %s", result.Status)
	}

	if result.Error != "terraform init failed" {
		t.Errorf("expected error message, got %s", result.Error)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
