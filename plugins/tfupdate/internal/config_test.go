package tfupdateengine

import (
	"testing"
	"time"
)

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

func TestUpdateConfig_CacheDefaults(t *testing.T) {
	cfg := &UpdateConfig{}

	if got := cfg.CacheBackend(); got != DefaultCacheBackend {
		t.Fatalf("CacheBackend() = %q, want %q", got, DefaultCacheBackend)
	}
	if got := cfg.CacheNamespace(); got != DefaultCacheNamespace {
		t.Fatalf("CacheNamespace() = %q, want %q", got, DefaultCacheNamespace)
	}
	if got := cfg.CacheTTL(); got != DefaultCacheTTL {
		t.Fatalf("CacheTTL() = %v, want %v", got, DefaultCacheTTL)
	}
}

func TestUpdateConfig_CacheOverrides(t *testing.T) {
	cfg := &UpdateConfig{
		Cache: &CacheConfig{
			Backend:   "redis",
			Namespace: "custom/update",
			TTL:       "2h",
		},
	}

	if got := cfg.CacheBackend(); got != "redis" {
		t.Fatalf("CacheBackend() = %q, want %q", got, "redis")
	}
	if got := cfg.CacheNamespace(); got != "custom/update" {
		t.Fatalf("CacheNamespace() = %q, want %q", got, "custom/update")
	}
	if got := cfg.CacheTTL(); got != 2*time.Hour {
		t.Fatalf("CacheTTL() = %v, want %v", got, 2*time.Hour)
	}
}

func TestUpdateConfig_ValidateTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout string
		wantErr bool
	}{
		{name: "empty", timeout: "", wantErr: false},
		{name: "valid", timeout: "15m", wantErr: false},
		{name: "invalid", timeout: "later", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &UpdateConfig{Timeout: tt.timeout}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateConfig_CommandTimeout(t *testing.T) {
	tests := []struct {
		name  string
		cfg   *UpdateConfig
		write bool
		want  time.Duration
	}{
		{name: "read default", cfg: &UpdateConfig{}, write: false, want: DefaultReadTimeout},
		{name: "write default", cfg: &UpdateConfig{}, write: true, want: DefaultWriteTimeout},
		{name: "configured override", cfg: &UpdateConfig{Timeout: "45m"}, write: true, want: 45 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.CommandTimeout(tt.write); got != tt.want {
				t.Fatalf("CommandTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}
