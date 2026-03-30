package updateengine

import "testing"

func TestApplyUpdates_ErrorSetsError(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Dependency: ModuleDependency{CallName: "vpc", Constraint: "~> 5.0"}, Status: StatusUpdateAvailable, File: "/nonexistent/file.tf", BumpedVersion: "5.2.0"},
		},
		Providers: []ProviderVersionUpdate{
			{Dependency: ProviderDependency{ProviderName: "aws", Constraint: "~> 5.0"}, Status: StatusUpdateAvailable, File: "/nonexistent/file.tf", BumpedVersion: "5.2.0"},
		},
	}

	NewApplyService().Apply(result)

	if result.Modules[0].Status != StatusError {
		t.Errorf("Module.Status = %q, want %q", result.Modules[0].Status, StatusError)
	}
	if result.Modules[0].Issue == "" {
		t.Error("Module.Issue should be set after write error")
	}
	if result.Providers[0].Status != StatusError {
		t.Errorf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusError)
	}
	if result.Providers[0].Issue == "" {
		t.Error("Provider.Issue should be set after write error")
	}
}

func TestApplyUpdates_SkipsNotUpdated(_ *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Status: StatusUpToDate, File: "some.tf"},
		},
		Providers: []ProviderVersionUpdate{
			{Status: StatusUpdateAvailable, File: ""},
		},
	}

	NewApplyService().Apply(result)
}

func TestParseVersionOrZero(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got := parseVersionOrZero("1.2.3")
		if got != (Version{1, 2, 3, ""}) {
			t.Errorf("parseVersionOrZero(1.2.3) = %v", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		got := parseVersionOrZero("bad")
		if !got.IsZero() {
			t.Errorf("parseVersionOrZero(bad) = %v, want zero", got)
		}
	})
}

func TestComputeSummary(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Status: StatusUpdateAvailable},
			{Status: StatusSkipped},
			{Status: StatusUpToDate},
		},
		Providers: []ProviderVersionUpdate{
			{Status: StatusApplied},
			{Status: StatusError},
		},
	}

	s := BuildUpdateSummary(result)
	if s.TotalChecked != 5 {
		t.Errorf("TotalChecked = %d, want 5", s.TotalChecked)
	}
	if s.UpdatesAvailable != 1 {
		t.Errorf("UpdatesAvailable = %d, want 1", s.UpdatesAvailable)
	}
	if s.UpdatesApplied != 1 {
		t.Errorf("UpdatesApplied = %d, want 1", s.UpdatesApplied)
	}
	if s.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", s.Skipped)
	}
	if s.Errors != 1 {
		t.Errorf("Errors = %d, want 1", s.Errors)
	}
}
