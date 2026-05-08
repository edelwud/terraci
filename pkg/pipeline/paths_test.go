package pipeline

import "testing"

func TestCanonicalPlanPathsUseWorkspaceSeparators(t *testing.T) {
	t.Parallel()

	modulePath := "svc\\prod\\eu\\vpc"
	if got := PlanBinaryPath(modulePath); got != "svc/prod/eu/vpc/plan.tfplan" {
		t.Fatalf("PlanBinaryPath() = %q", got)
	}
	if got := PlanTextPath(modulePath); got != "svc/prod/eu/vpc/plan.txt" {
		t.Fatalf("PlanTextPath() = %q", got)
	}
	if got := PlanJSONPath(modulePath); got != "svc/prod/eu/vpc/plan.json" {
		t.Fatalf("PlanJSONPath() = %q", got)
	}
}

func TestWorkspacePathDoesNotHideParentSegments(t *testing.T) {
	t.Parallel()

	got := WorkspacePath("svc/../vpc", PlanJSONFilename)
	if got != "svc/../vpc/plan.json" {
		t.Fatalf("WorkspacePath() = %q, want parent segment preserved for validation", got)
	}
	if err := ValidateWorkspacePath(got); err == nil {
		t.Fatal("ValidateWorkspacePath() error = nil, want parent segment rejected")
	}
}

func TestValidateWorkspacePath(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"svc/prod/eu/vpc/plan.json",
		".terraci/cost-report.json",
		`svc\prod\eu\vpc\plan.json`,
	} {
		if err := ValidateWorkspacePath(path); err != nil {
			t.Fatalf("ValidateWorkspacePath(%q) error = %v", path, err)
		}
	}

	for _, path := range []string{
		"",
		"/tmp/plan.json",
		"../plan.json",
		"svc/../plan.json",
		`C:\tmp\plan.json`,
	} {
		if err := ValidateWorkspacePath(path); err == nil {
			t.Fatalf("ValidateWorkspacePath(%q) error = nil, want error", path)
		}
	}
}
