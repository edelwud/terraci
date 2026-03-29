package updateengine

import "testing"

func TestUpdateConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		bump    string
		wantErr bool
	}{
		{"empty defaults", "", "", false},
		{"all+patch", "all", "patch", false},
		{"modules+minor", "modules", "minor", false},
		{"providers+major", "providers", "major", false},
		{"invalid target", "invalid", "", true},
		{"invalid bump", "", "huge", true},
		{"both invalid", "bad", "bad", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &UpdateConfig{Target: tt.target, Bump: tt.bump}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateConfig_ShouldCheckProviders(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		{TargetAll, true},
		{TargetProviders, true},
		{"", true},
		{TargetModules, false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			cfg := &UpdateConfig{Target: tt.target}
			if got := cfg.ShouldCheckProviders(); got != tt.want {
				t.Errorf("ShouldCheckProviders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateConfig_ShouldCheckModules(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		{TargetAll, true},
		{TargetModules, true},
		{"", true},
		{TargetProviders, false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			cfg := &UpdateConfig{Target: tt.target}
			if got := cfg.ShouldCheckModules(); got != tt.want {
				t.Errorf("ShouldCheckModules() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateConfig_IsIgnored(t *testing.T) {
	tests := []struct {
		name   string
		ignore []string
		source string
		want   bool
	}{
		{"in list", []string{"hashicorp/aws"}, "hashicorp/aws", true},
		{"not in list", []string{"hashicorp/aws"}, "hashicorp/gcp", false},
		{"empty list", []string{}, "hashicorp/aws", false},
		{"nil list", nil, "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &UpdateConfig{Ignore: tt.ignore}
			if got := cfg.IsIgnored(tt.source); got != tt.want {
				t.Errorf("IsIgnored(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}
