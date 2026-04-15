package domain

import "testing"

func TestModuleVersionUpdateStatusHelpers(t *testing.T) {
	update := NewModuleVersionUpdate(ModuleDependency{
		ModulePath: "svc/prod/app",
		CallName:   "vpc",
		Constraint: "~> 5.0",
	})

	if update.Status != StatusUpToDate {
		t.Fatalf("Status = %q, want %q", update.Status, StatusUpToDate)
	}
	if update.IsApplyPending() {
		t.Fatal("new update should not be apply pending")
	}

	update.Status = StatusUpdateAvailable
	if !update.IsApplyPending() {
		t.Fatal("update should be apply pending")
	}
	if got := update.MarkApplied(); got.Status != StatusApplied || got.Issue != "" {
		t.Fatalf("MarkApplied() = %#v", got)
	}
}

func TestProviderVersionUpdateDisplayHelpers(t *testing.T) {
	update := ProviderVersionUpdate{
		Dependency:     ProviderDependency{Constraint: "~> 5.0"},
		CurrentVersion: "5.1.0",
		LatestVersion:  "5.2.0",
		BumpedVersion:  "5.2.0",
	}

	if got := update.DisplayCurrent(); got != "~> 5.0 (5.1.0)" {
		t.Fatalf("DisplayCurrent() = %q", got)
	}
	if got := update.DisplayLatest(); got != "" {
		t.Fatalf("DisplayLatest() = %q, want empty when latest equals bumped", got)
	}
}
