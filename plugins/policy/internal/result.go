package policyengine

import (
	"encoding/json"
	"sort"
)

type Status string

type Namespace string

type Namespaces []Namespace

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

func (s Status) String() string {
	return string(s)
}

func (n Namespace) String() string {
	return string(n)
}

func NewNamespaces(values []string) Namespaces {
	if len(values) == 0 {
		return nil
	}
	out := make(Namespaces, len(values))
	for i, value := range values {
		out[i] = Namespace(value)
	}
	return out
}

func (n Namespaces) Clone() Namespaces {
	return append(Namespaces(nil), n...)
}

func (n Namespaces) Strings() []string {
	if len(n) == 0 {
		return nil
	}
	out := make([]string, len(n))
	for i, namespace := range n {
		out[i] = namespace.String()
	}
	return out
}

type FindingMetadata struct {
	values map[string]any
}

// Finding represents one policy decision produced by a Rego rule.
type Finding struct {
	Message   string
	Namespace Namespace
	Metadata  FindingMetadata
}

// Evaluation is the raw OPA decision set before TerraCi enforcement actions
// turn decisions into blocking failures, warnings, or ignored findings.
type Evaluation struct {
	denies []Finding
	warns  []Finding
}

func NewFinding(namespace Namespace, message string, metadata FindingMetadata) Finding {
	return Finding{
		Message:   message,
		Namespace: namespace,
		Metadata:  metadata.Clone(),
	}
}

func NewEvaluation(denies, warns []Finding) *Evaluation {
	return &Evaluation{
		denies: cloneFindings(denies),
		warns:  cloneFindings(warns),
	}
}

func EmptyEvaluation() *Evaluation {
	return &Evaluation{}
}

func (e *Evaluation) Denies() []Finding {
	if e == nil {
		return nil
	}
	return cloneFindings(e.denies)
}

func (e *Evaluation) Warns() []Finding {
	if e == nil {
		return nil
	}
	return cloneFindings(e.warns)
}

func NewFindingMetadata(values map[string]any) FindingMetadata {
	return FindingMetadata{values: cloneAnyMap(values)}
}

func (m FindingMetadata) Clone() FindingMetadata {
	return NewFindingMetadata(m.values)
}

func (m FindingMetadata) Map() map[string]any {
	return cloneAnyMap(m.values)
}

func (m FindingMetadata) Empty() bool {
	return len(m.values) == 0
}

func (m FindingMetadata) MarshalJSON() ([]byte, error) {
	if len(m.values) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(m.values)
}

func (f Finding) MarshalJSON() ([]byte, error) {
	payload := struct {
		Message   string           `json:"msg"`
		Namespace Namespace        `json:"namespace,omitempty"`
		Metadata  *FindingMetadata `json:"metadata,omitempty"`
	}{
		Message:   f.Message,
		Namespace: f.Namespace,
	}
	if !f.Metadata.Empty() {
		metadata := f.Metadata.Clone()
		payload.Metadata = &metadata
	}
	return json.Marshal(payload)
}

func (f *Finding) UnmarshalJSON(data []byte) error {
	var payload struct {
		Message   string         `json:"msg"`
		Namespace Namespace      `json:"namespace,omitempty"`
		Metadata  map[string]any `json:"metadata,omitempty"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	f.Message = payload.Message
	f.Namespace = payload.Namespace
	f.Metadata = NewFindingMetadata(payload.Metadata)
	return nil
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
			Namespace: Namespace("terraci"),
		}},
	}
}

func ApplyEvaluation(modulePath string, eval *Evaluation, decisions Decisions) Result {
	result := Result{Module: modulePath}
	if eval == nil {
		return result
	}

	decisions = decisions.Normalize()
	applyFindings(&result, eval.denies, decisions.Deny)
	applyFindings(&result, eval.warns, decisions.Warn)
	sortFindings(result.Failures)
	sortFindings(result.Warnings)
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

func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Namespace == findings[j].Namespace {
			return findings[i].Message < findings[j].Message
		}
		return findings[i].Namespace < findings[j].Namespace
	})
}

func (r *Result) HasFailures() bool {
	return len(r.Failures) > 0
}

func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}

func (r *Result) Status() Status {
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
	ordered := append([]Result(nil), results...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Module < ordered[j].Module
	})

	s := &Summary{
		TotalModules: len(ordered),
		Results:      ordered,
	}

	for _, r := range ordered {
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
		case StatusPass:
			s.PassedModules++
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

func (s *Summary) Status() Status {
	if s.HasFailures() {
		return StatusFail
	}
	if s.HasWarnings() {
		return StatusWarn
	}
	return StatusPass
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneAnyValue(value)
	}
	return out
}

func cloneFindings(in []Finding) []Finding {
	if len(in) == 0 {
		return nil
	}
	out := make([]Finding, len(in))
	for i, finding := range in {
		out[i] = finding
		out[i].Metadata = finding.Metadata.Clone()
	}
	return out
}

func cloneAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneAnyValue(item)
		}
		return out
	default:
		return typed
	}
}
