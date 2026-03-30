package updateengine

import "testing"

func TestApplyUpdates_ErrorSetsError(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{UpdateAvailable: true, File: "/nonexistent/file.tf", CallName: "vpc", BumpedVersion: "5.2.0", Constraint: "~> 5.0"},
		},
		Providers: []ProviderVersionUpdate{
			{UpdateAvailable: true, File: "/nonexistent/file.tf", ProviderName: "aws", BumpedVersion: "5.2.0", Constraint: "~> 5.0"},
		},
	}

	NewApplyService().Apply(result)

	if result.Modules[0].Error == "" {
		t.Error("Module.Error should be set after write error")
	}
	if result.Providers[0].Error == "" {
		t.Error("Provider.Error should be set after write error")
	}
}

func TestApplyUpdates_SkipsNotUpdated(_ *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{UpdateAvailable: false, File: "some.tf"},
		},
		Providers: []ProviderVersionUpdate{
			{UpdateAvailable: true, File: ""},
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
			{UpdateAvailable: true},
			{Skipped: true},
			{},
		},
		Providers: []ProviderVersionUpdate{
			{UpdateAvailable: true},
			{Skipped: true},
		},
	}

	s := BuildUpdateSummary(result)
	if s.TotalChecked != 5 {
		t.Errorf("TotalChecked = %d, want 5", s.TotalChecked)
	}
	if s.UpdatesAvailable != 2 {
		t.Errorf("UpdatesAvailable = %d, want 2", s.UpdatesAvailable)
	}
	if s.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", s.Skipped)
	}
}
