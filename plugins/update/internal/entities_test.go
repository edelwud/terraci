package updateengine

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
