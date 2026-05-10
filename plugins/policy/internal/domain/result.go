package domain

const (
	StatusPass = "pass"
	StatusWarn = "warn"
	StatusFail = "fail"
)

// Finding represents one policy decision produced by a Rego rule.
type Finding struct {
	Message   string         `json:"msg"`
	Namespace string         `json:"namespace,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Evaluation is the raw OPA decision set before TerraCi enforcement actions
// turn decisions into blocking failures, warnings, or ignored findings.
type Evaluation struct {
	Denies []Finding
	Warns  []Finding
}

// Result represents the policy check result for a single module.
type Result struct {
	Module     string    `json:"module"`
	Failures   []Finding `json:"failures,omitempty"`
	Warnings   []Finding `json:"warnings,omitempty"`
	Successes  int       `json:"successes"`
	Skipped    int       `json:"skipped"`
	Suppressed int       `json:"suppressed,omitempty"`
}

func NewSkippedResult(modulePath string) Result {
	return Result{Module: modulePath, Skipped: 1}
}

func NewErrorResult(modulePath string, err error) Result {
	return Result{
		Module: modulePath,
		Failures: []Finding{{
			Message:   "policy check failed: " + err.Error(),
			Namespace: "terraci",
		}},
	}
}

func ApplyEvaluation(modulePath string, eval *Evaluation, policy ActionPolicy) Result {
	result := Result{Module: modulePath}
	if eval == nil {
		return result
	}

	applyFindings(&result, eval.Denies, policy.FailureAction)
	applyFindings(&result, eval.Warns, policy.WarningAction)
	return result
}

func applyFindings(result *Result, findings []Finding, action Action) {
	switch action {
	case ActionBlock:
		result.Failures = append(result.Failures, findings...)
	case ActionWarn:
		result.Warnings = append(result.Warnings, findings...)
	case ActionIgnore:
		result.Suppressed += len(findings)
	default:
		result.Failures = append(result.Failures, findings...)
	}
}

func (r *Result) HasFailures() bool {
	return len(r.Failures) > 0
}

func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}

func (r *Result) Status() string {
	if r.HasFailures() {
		return StatusFail
	}
	if r.HasWarnings() {
		return StatusWarn
	}
	return StatusPass
}

// Summary aggregates results from multiple modules.
type Summary struct {
	TotalModules    int      `json:"total_modules"`
	PassedModules   int      `json:"passed_modules"`
	WarnedModules   int      `json:"warned_modules"`
	FailedModules   int      `json:"failed_modules"`
	SkippedModules  int      `json:"skipped_modules,omitempty"`
	TotalFailures   int      `json:"total_failures"`
	TotalWarnings   int      `json:"total_warnings"`
	TotalSuppressed int      `json:"total_suppressed,omitempty"`
	Results         []Result `json:"results"`
}

func NewSummary(results []Result) *Summary {
	s := &Summary{
		TotalModules: len(results),
		Results:      append([]Result(nil), results...),
	}

	for _, r := range results {
		s.TotalFailures += len(r.Failures)
		s.TotalWarnings += len(r.Warnings)
		s.TotalSuppressed += r.Suppressed
		if r.Skipped > 0 {
			s.SkippedModules++
		}

		switch r.Status() {
		case StatusFail:
			s.FailedModules++
		case StatusWarn:
			s.WarnedModules++
		default:
			s.PassedModules++
		}
	}

	return s
}

func (s *Summary) HasFailures() bool {
	return s != nil && s.FailedModules > 0
}

func (s *Summary) HasWarnings() bool {
	return s != nil && (s.WarnedModules > 0 || s.TotalWarnings > 0)
}

func (s *Summary) Status() string {
	if s.HasFailures() {
		return StatusFail
	}
	if s.HasWarnings() {
		return StatusWarn
	}
	return StatusPass
}
