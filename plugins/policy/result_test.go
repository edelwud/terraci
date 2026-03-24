package policy

import (
	"testing"
)

func TestResult_HasFailures(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		expected bool
	}{
		{
			name:     "no failures",
			result:   Result{},
			expected: false,
		},
		{
			name: "with failures",
			result: Result{
				Failures: []Violation{{Message: "test failure"}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasFailures(); got != tt.expected {
				t.Errorf("HasFailures() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResult_HasWarnings(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		expected bool
	}{
		{
			name:     "no warnings",
			result:   Result{},
			expected: false,
		},
		{
			name: "with warnings",
			result: Result{
				Warnings: []Violation{{Message: "test warning"}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasWarnings(); got != tt.expected {
				t.Errorf("HasWarnings() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResult_Status(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		expected string
	}{
		{
			name:     "pass when no issues",
			result:   Result{Successes: 1},
			expected: StatusPass,
		},
		{
			name: "warn when only warnings",
			result: Result{
				Warnings: []Violation{{Message: "warning"}},
			},
			expected: StatusWarn,
		},
		{
			name: "fail when has failures",
			result: Result{
				Failures: []Violation{{Message: "failure"}},
			},
			expected: StatusFail,
		},
		{
			name: "fail takes precedence over warn",
			result: Result{
				Failures: []Violation{{Message: "failure"}},
				Warnings: []Violation{{Message: "warning"}},
			},
			expected: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Status(); got != tt.expected {
				t.Errorf("Status() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewSummary(t *testing.T) {
	results := []Result{
		{Module: "mod1", Successes: 1},
		{Module: "mod2", Warnings: []Violation{{Message: "w1"}, {Message: "w2"}}},
		{Module: "mod3", Failures: []Violation{{Message: "f1"}}},
	}

	summary := NewSummary(results)

	if summary.TotalModules != 3 {
		t.Errorf("TotalModules = %v, want %v", summary.TotalModules, 3)
	}
	if summary.PassedModules != 1 {
		t.Errorf("PassedModules = %v, want %v", summary.PassedModules, 1)
	}
	if summary.WarnedModules != 1 {
		t.Errorf("WarnedModules = %v, want %v", summary.WarnedModules, 1)
	}
	if summary.FailedModules != 1 {
		t.Errorf("FailedModules = %v, want %v", summary.FailedModules, 1)
	}
	if summary.TotalFailures != 1 {
		t.Errorf("TotalFailures = %v, want %v", summary.TotalFailures, 1)
	}
	if summary.TotalWarnings != 2 {
		t.Errorf("TotalWarnings = %v, want %v", summary.TotalWarnings, 2)
	}
}

func TestSummary_HasFailures(t *testing.T) {
	tests := []struct {
		name     string
		summary  Summary
		expected bool
	}{
		{
			name:     "no failures",
			summary:  Summary{PassedModules: 1},
			expected: false,
		},
		{
			name:     "has failed modules",
			summary:  Summary{FailedModules: 1},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summary.HasFailures(); got != tt.expected {
				t.Errorf("HasFailures() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSummary_HasWarnings(t *testing.T) {
	tests := []struct {
		name     string
		summary  Summary
		expected bool
	}{
		{
			name:     "no warnings",
			summary:  Summary{PassedModules: 1},
			expected: false,
		},
		{
			name:     "has warned modules",
			summary:  Summary{WarnedModules: 1},
			expected: true,
		},
		{
			name:     "has total warnings",
			summary:  Summary{TotalWarnings: 1},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summary.HasWarnings(); got != tt.expected {
				t.Errorf("HasWarnings() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSummary_Status(t *testing.T) {
	tests := []struct {
		name     string
		summary  Summary
		expected string
	}{
		{
			name:     "pass when all passed",
			summary:  Summary{PassedModules: 3},
			expected: StatusPass,
		},
		{
			name:     "warn when has warnings",
			summary:  Summary{WarnedModules: 1},
			expected: StatusWarn,
		},
		{
			name:     "fail when has failures",
			summary:  Summary{FailedModules: 1},
			expected: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summary.Status(); got != tt.expected {
				t.Errorf("Status() = %v, want %v", got, tt.expected)
			}
		})
	}
}
