package config

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/plugins/policy/internal/domain"
)

func boolPtr(v bool) *bool { return &v }

func TestConfig_ValidateNewSourceShape(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Sources: []SourceConfig{
			{Type: SourceTypePath, Path: "policies"},
			{Type: SourceTypeGit, URL: "https://github.com/org/policies.git", Ref: "main"},
			{Type: SourceTypeOCI, URL: "oci://ghcr.io/org/policies:v1"},
		},
		FailureAction: domain.ActionBlock,
		WarningAction: domain.ActionWarn,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestConfig_RejectsLegacyFields(t *testing.T) {
	t.Parallel()

	tests := []string{
		"enabled: true\non_failure: warn\nsources: [{type: path, path: policies}]\n",
		"enabled: true\non_warning: ignore\nsources: [{type: path, path: policies}]\n",
		"enabled: true\noverwrites: []\nsources: [{type: path, path: policies}]\n",
		"enabled: true\nsources: [{git: https://github.com/org/policies.git, ref: main}]\n",
	}

	for _, data := range tests {
		t.Run(strings.Split(data, "\n")[1], func(t *testing.T) {
			t.Parallel()

			var cfg Config
			err := yaml.Unmarshal([]byte(data), &cfg)
			if err == nil {
				t.Fatal("Unmarshal() error = nil, want legacy field rejection")
			}
			if !strings.Contains(err.Error(), "no longer supported") {
				t.Fatalf("error = %q, want legacy field message", err.Error())
			}
		})
	}
}

func TestConfig_EffectiveConfigAppliesOverridesInOrder(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enabled:       true,
		Sources:       []SourceConfig{{Type: SourceTypePath, Path: "policies"}},
		Namespaces:    []string{"terraform"},
		FailureAction: domain.ActionBlock,
		WarningAction: domain.ActionWarn,
		Overrides: []Override{
			{
				Match:         "**/prod/**",
				Enabled:       boolPtr(false),
				Namespaces:    []string{"terraform", "audit"},
				FailureAction: domain.ActionWarn,
			},
			{
				Match:         "platform/**",
				Enabled:       boolPtr(true),
				WarningAction: domain.ActionIgnore,
			},
		},
	}

	effective, err := cfg.EffectiveConfig("platform/prod/eu-central-1/app")
	if err != nil {
		t.Fatalf("EffectiveConfig() error = %v", err)
	}
	if !effective.Enabled {
		t.Fatal("Enabled = false, want later override to re-enable")
	}
	if effective.FailureAction != domain.ActionWarn {
		t.Fatalf("FailureAction = %q, want warn", effective.FailureAction)
	}
	if effective.WarningAction != domain.ActionIgnore {
		t.Fatalf("WarningAction = %q, want ignore", effective.WarningAction)
	}
	if got := strings.Join(effective.Namespaces, ","); got != "terraform,audit" {
		t.Fatalf("Namespaces = %q, want terraform,audit", got)
	}
}
