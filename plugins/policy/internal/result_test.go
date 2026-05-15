package policyengine

import "testing"

func TestApplyEvaluation_ActionMapping(t *testing.T) {
	t.Parallel()

	eval := &Evaluation{
		Denies: []Finding{{Message: "deny"}},
		Warns:  []Finding{{Message: "warn"}},
	}

	tests := []struct {
		name           string
		decisions      Decisions
		wantFailures   int
		wantWarnings   int
		wantSuppressed int
		wantStatus     string
	}{
		{
			name:         "block deny warn warning",
			decisions:    Decisions{Deny: ActionBlock, Warn: ActionWarn},
			wantFailures: 1,
			wantWarnings: 1,
			wantStatus:   StatusFail,
		},
		{
			name:           "warn deny ignore warning",
			decisions:      Decisions{Deny: ActionWarn, Warn: ActionIgnore},
			wantWarnings:   1,
			wantSuppressed: 1,
			wantStatus:     StatusWarn,
		},
		{
			name:           "ignore deny block warning",
			decisions:      Decisions{Deny: ActionIgnore, Warn: ActionBlock},
			wantFailures:   1,
			wantSuppressed: 1,
			wantStatus:     StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ApplyEvaluation("platform/prod/app", eval, tt.decisions)
			if len(result.Failures) != tt.wantFailures {
				t.Fatalf("failures = %d, want %d", len(result.Failures), tt.wantFailures)
			}
			if len(result.Warnings) != tt.wantWarnings {
				t.Fatalf("warnings = %d, want %d", len(result.Warnings), tt.wantWarnings)
			}
			if result.Suppressed != tt.wantSuppressed {
				t.Fatalf("suppressed = %d, want %d", result.Suppressed, tt.wantSuppressed)
			}
			if result.Status() != tt.wantStatus {
				t.Fatalf("status = %q, want %q", result.Status(), tt.wantStatus)
			}
		})
	}
}

func TestNewSummary_CountsSkippedAndSuppressed(t *testing.T) {
	t.Parallel()

	summary := NewSummary([]Result{
		{Module: "pass"},
		{Module: "skip", Skipped: 1},
		{Module: "warn", Warnings: []Finding{{Message: "warn"}}, Suppressed: 2},
		{Module: "fail", Failures: []Finding{{Message: "fail"}}},
	})

	if summary.TotalModules != 4 {
		t.Fatalf("TotalModules = %d, want 4", summary.TotalModules)
	}
	if summary.PassedModules != 2 {
		t.Fatalf("PassedModules = %d, want 2", summary.PassedModules)
	}
	if summary.SkippedModules != 1 {
		t.Fatalf("SkippedModules = %d, want 1", summary.SkippedModules)
	}
	if summary.WarnedModules != 1 || summary.FailedModules != 1 {
		t.Fatalf("warned/failed = %d/%d, want 1/1", summary.WarnedModules, summary.FailedModules)
	}
	if summary.TotalSuppressed != 2 {
		t.Fatalf("TotalSuppressed = %d, want 2", summary.TotalSuppressed)
	}
}
