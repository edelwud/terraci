package tfupdateengine

import "testing"

func TestModuleVersionUpdate_StatusHelpers(t *testing.T) {
	tests := []struct {
		name          string
		update        ModuleVersionUpdate
		wantLabel     string
		wantUpdatable bool
	}{
		{
			name:          "up to date",
			update:        ModuleVersionUpdate{Status: StatusUpToDate},
			wantLabel:     "up to date",
			wantUpdatable: false,
		},
		{
			name:          "update available",
			update:        ModuleVersionUpdate{Status: StatusUpdateAvailable},
			wantLabel:     "update available",
			wantUpdatable: true,
		},
		{
			name:          "applied",
			update:        ModuleVersionUpdate{Status: StatusApplied},
			wantLabel:     "applied",
			wantUpdatable: true,
		},
		{
			name:          "skipped",
			update:        ModuleVersionUpdate{Status: StatusSkipped, Issue: "ignored by config"},
			wantLabel:     "skipped: ignored by config",
			wantUpdatable: false,
		},
		{
			name:          "error",
			update:        ModuleVersionUpdate{Status: StatusError, Issue: "registry error: timeout"},
			wantLabel:     "error: registry error: timeout",
			wantUpdatable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.update.StatusLabel(); got != tt.wantLabel {
				t.Errorf("StatusLabel() = %q, want %q", got, tt.wantLabel)
			}
			if got := tt.update.IncludedInUpdateLogs(); got != tt.wantUpdatable {
				t.Errorf("IncludedInUpdateLogs() = %v, want %v", got, tt.wantUpdatable)
			}
		})
	}
}

func TestProviderVersionUpdate_StatusHelpers(t *testing.T) {
	tests := []struct {
		name          string
		update        ProviderVersionUpdate
		wantLabel     string
		wantUpdatable bool
	}{
		{
			name:          "up to date",
			update:        ProviderVersionUpdate{Status: StatusUpToDate},
			wantLabel:     "up to date",
			wantUpdatable: false,
		},
		{
			name:          "update available",
			update:        ProviderVersionUpdate{Status: StatusUpdateAvailable},
			wantLabel:     "update available",
			wantUpdatable: true,
		},
		{
			name:          "applied",
			update:        ProviderVersionUpdate{Status: StatusApplied},
			wantLabel:     "applied",
			wantUpdatable: true,
		},
		{
			name:          "skipped",
			update:        ProviderVersionUpdate{Status: StatusSkipped, Issue: "ignored by config"},
			wantLabel:     "skipped: ignored by config",
			wantUpdatable: false,
		},
		{
			name:          "error",
			update:        ProviderVersionUpdate{Status: StatusError, Issue: "registry error: timeout"},
			wantLabel:     "error: registry error: timeout",
			wantUpdatable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.update.StatusLabel(); got != tt.wantLabel {
				t.Errorf("StatusLabel() = %q, want %q", got, tt.wantLabel)
			}
			if got := tt.update.IncludedInUpdateLogs(); got != tt.wantUpdatable {
				t.Errorf("IncludedInUpdateLogs() = %v, want %v", got, tt.wantUpdatable)
			}
		})
	}
}

func TestNewModuleVersionUpdate(t *testing.T) {
	dep := ModuleDependency{
		ModulePath: "platform/prod/vpc",
		CallName:   "vpc",
		Source:     "terraform-aws-modules/vpc/aws",
		Constraint: "~> 5.0",
	}

	update := NewModuleVersionUpdate(dep)
	if update.ModulePath() != dep.ModulePath {
		t.Errorf("ModulePath = %q, want %q", update.ModulePath(), dep.ModulePath)
	}
	if update.CallName() != dep.CallName {
		t.Errorf("CallName = %q, want %q", update.CallName(), dep.CallName)
	}
	if update.Source() != dep.Source {
		t.Errorf("Source = %q, want %q", update.Source(), dep.Source)
	}
	if update.Constraint() != dep.Constraint {
		t.Errorf("Constraint = %q, want %q", update.Constraint(), dep.Constraint)
	}
	if update.Status != StatusUpToDate {
		t.Errorf("Status = %q, want %q", update.Status, StatusUpToDate)
	}
}

func TestNewProviderVersionUpdate(t *testing.T) {
	dep := ProviderDependency{
		ModulePath:     "platform/prod/vpc",
		ProviderName:   "aws",
		ProviderSource: "hashicorp/aws",
		Constraint:     "~> 5.0",
	}

	update := NewProviderVersionUpdate(dep)
	if update.ModulePath() != dep.ModulePath {
		t.Errorf("ModulePath = %q, want %q", update.ModulePath(), dep.ModulePath)
	}
	if update.ProviderName() != dep.ProviderName {
		t.Errorf("ProviderName = %q, want %q", update.ProviderName(), dep.ProviderName)
	}
	if update.ProviderSource() != dep.ProviderSource {
		t.Errorf("ProviderSource = %q, want %q", update.ProviderSource(), dep.ProviderSource)
	}
	if update.Constraint() != dep.Constraint {
		t.Errorf("Constraint = %q, want %q", update.Constraint(), dep.Constraint)
	}
	if update.Status != StatusUpToDate {
		t.Errorf("Status = %q, want %q", update.Status, StatusUpToDate)
	}
}

func TestModuleVersionUpdate_DisplayHelpers(t *testing.T) {
	update := ModuleVersionUpdate{
		Dependency:     ModuleDependency{Constraint: "~> 5.0"},
		CurrentVersion: "5.84.0",
		LatestVersion:  "6.0.0",
		BumpedVersion:  "5.90.0",
		Status:         StatusApplied,
	}

	if got := update.DisplayCurrent(); got != "~> 5.0 (5.84.0)" {
		t.Errorf("DisplayCurrent() = %q, want %q", got, "~> 5.0 (5.84.0)")
	}
	if got := update.DisplayAvailable(); got != "5.90.0" {
		t.Errorf("DisplayAvailable() = %q, want %q", got, "5.90.0")
	}
	if got := update.DisplayLatest(); got != "6.0.0" {
		t.Errorf("DisplayLatest() = %q, want %q", got, "6.0.0")
	}
	if !update.IsApplied() {
		t.Error("IsApplied() = false, want true")
	}
}

func TestProviderVersionUpdate_DisplayHelpers(t *testing.T) {
	update := ProviderVersionUpdate{
		Dependency:     ProviderDependency{Constraint: "~> 5.0"},
		CurrentVersion: "5.84.0",
		LatestVersion:  "5.90.0",
		BumpedVersion:  "5.90.0",
		Status:         StatusUpdateAvailable,
	}

	if got := update.DisplayCurrent(); got != "~> 5.0 (5.84.0)" {
		t.Errorf("DisplayCurrent() = %q, want %q", got, "~> 5.0 (5.84.0)")
	}
	if got := update.DisplayAvailable(); got != "5.90.0" {
		t.Errorf("DisplayAvailable() = %q, want %q", got, "5.90.0")
	}
	if got := update.DisplayLatest(); got != "" {
		t.Errorf("DisplayLatest() = %q, want empty", got)
	}
	if update.IsApplied() {
		t.Error("IsApplied() = true, want false")
	}
}
