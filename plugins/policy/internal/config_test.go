package policyengine

import (
	"strings"
	"testing"
)

func boolPtr(v bool) *bool { return &v }

func TestConfig_GetEffectiveConfigAppliesAllMatchingOverwritesInOrder(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enabled:    true,
		OnFailure:  ActionBlock,
		OnWarning:  ActionWarn,
		Namespaces: []string{"terraform"},
		Overwrites: []Overwrite{
			{
				Match:      "**/prod/**",
				Enabled:    boolPtr(false),
				OnFailure:  ActionWarn,
				Namespaces: []string{"terraform", "audit"},
			},
			{
				Match:     "platform/**",
				Enabled:   boolPtr(true),
				OnFailure: ActionBlock,
				OnWarning: ActionIgnore,
			},
		},
	}

	effective, err := cfg.GetEffectiveConfig("platform/prod/eu-central-1/app")
	if err != nil {
		t.Fatalf("GetEffectiveConfig() error = %v", err)
	}
	if effective == nil {
		t.Fatal("GetEffectiveConfig() = nil")
	}
	if !effective.Enabled {
		t.Fatal("Enabled = false, want true from later overwrite")
	}
	if effective.OnFailure != ActionBlock {
		t.Fatalf("OnFailure = %q, want block", effective.OnFailure)
	}
	if effective.OnWarning != ActionIgnore {
		t.Fatalf("OnWarning = %q, want ignore", effective.OnWarning)
	}
	if got := strings.Join(effective.Namespaces, ","); got != "terraform,audit" {
		t.Fatalf("Namespaces = %q, want terraform,audit", got)
	}
}

func TestConfig_GetEffectiveConfigReturnsMalformedGlobError(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enabled: true,
		Overwrites: []Overwrite{{
			Match: "platform/[bad/**",
		}},
	}

	_, err := cfg.GetEffectiveConfig("platform/prod/eu-central-1/app")
	if err == nil {
		t.Fatal("GetEffectiveConfig() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "overwrites[0]") {
		t.Fatalf("error = %q, want overwrite index", err)
	}
}

func TestConfig_ValidateRejectsMalformedOverwriteMatch(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Sources: []SourceConfig{{Path: "terraform"}},
		Overwrites: []Overwrite{{
			Match: "platform/[bad/**",
		}},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "overwrites[0].match") {
		t.Fatalf("error = %q, want overwrite match path", err)
	}
}
