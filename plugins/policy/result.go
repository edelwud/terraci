// Package policy provides OPA-based policy checking for Terraform plans
package policy

// Status constants for policy check results
const (
	StatusPass = "pass"
	StatusWarn = "warn"
	StatusFail = "fail"
)

// Result represents the policy check result for a single module
type Result struct {
	// Module is the module path (e.g., "platform/prod/eu-central-1/vpc")
	Module string `json:"module"`

	// Failures are policy violations that should block the pipeline
	Failures []Violation `json:"failures,omitempty"`

	// Warnings are policy violations that should be reported but not block
	Warnings []Violation `json:"warnings,omitempty"`

	// Successes is the count of passed policy rules
	Successes int `json:"successes"`

	// Skipped is the count of skipped policy rules
	Skipped int `json:"skipped"`
}

// Violation represents a single policy violation
type Violation struct {
	// Message is the violation message from the policy
	Message string `json:"msg"`

	// Namespace is the Rego package that produced this violation
	Namespace string `json:"namespace,omitempty"`

	// Metadata contains additional context from the policy
	Metadata map[string]any `json:"metadata,omitempty"`
}

// HasFailures returns true if there are any failures
func (r *Result) HasFailures() bool {
	return len(r.Failures) > 0
}

// HasWarnings returns true if there are any warnings
func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// Status returns the overall status: StatusPass, StatusWarn, or StatusFail
func (r *Result) Status() string {
	if r.HasFailures() {
		return StatusFail
	}
	if r.HasWarnings() {
		return StatusWarn
	}
	return StatusPass
}

// Summary aggregates results from multiple modules
type Summary struct {
	// TotalModules is the total number of modules checked
	TotalModules int `json:"total_modules"`

	// PassedModules is the number of modules that passed
	PassedModules int `json:"passed_modules"`

	// WarnedModules is the number of modules with warnings only
	WarnedModules int `json:"warned_modules"`

	// FailedModules is the number of modules that failed
	FailedModules int `json:"failed_modules"`

	// TotalFailures is the total number of failures across all modules
	TotalFailures int `json:"total_failures"`

	// TotalWarnings is the total number of warnings across all modules
	TotalWarnings int `json:"total_warnings"`

	// Results contains per-module results
	Results []Result `json:"results"`
}

// NewSummary creates a summary from a list of results
func NewSummary(results []Result) *Summary {
	s := &Summary{
		TotalModules: len(results),
		Results:      results,
	}

	for _, r := range results {
		s.TotalFailures += len(r.Failures)
		s.TotalWarnings += len(r.Warnings)

		switch r.Status() {
		case StatusPass:
			s.PassedModules++
		case StatusWarn:
			s.WarnedModules++
		case StatusFail:
			s.FailedModules++
		}
	}

	return s
}

// HasFailures returns true if any module has failures
func (s *Summary) HasFailures() bool {
	return s.FailedModules > 0
}

// HasWarnings returns true if any module has warnings
func (s *Summary) HasWarnings() bool {
	return s.WarnedModules > 0 || s.TotalWarnings > 0
}

// Status returns the overall status: StatusPass, StatusWarn, or StatusFail
func (s *Summary) Status() string {
	if s.HasFailures() {
		return StatusFail
	}
	if s.HasWarnings() {
		return StatusWarn
	}
	return StatusPass
}
