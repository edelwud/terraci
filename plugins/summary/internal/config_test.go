package summaryengine

import "testing"

func TestConfig_NormalizedOwnsMutableFields(t *testing.T) {
	enabled := true
	includeDetails := true
	cfg := &Config{
		Enabled:        &enabled,
		IncludeDetails: &includeDetails,
		Labels:         []string{"terraform"},
	}

	normalized := cfg.Normalized()
	enabled = false
	includeDetails = false
	cfg.Labels[0] = "changed"

	if normalized.Enabled == nil || !*normalized.Enabled {
		t.Fatalf("normalized Enabled = %v, want independent true pointer", normalized.Enabled)
	}
	if normalized.IncludeDetails == nil || !*normalized.IncludeDetails {
		t.Fatalf("normalized IncludeDetails = %v, want independent true pointer", normalized.IncludeDetails)
	}
	if got := normalized.Labels[0]; got != "terraform" {
		t.Fatalf("normalized Labels[0] = %q, want terraform", got)
	}
}
