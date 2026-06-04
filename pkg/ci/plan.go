package ci

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"sort"
	"strings"
	"time"
)

// PlanStatus represents the status of a terraform plan.
type PlanStatus string

const (
	PlanStatusPending   PlanStatus = "pending"
	PlanStatusRunning   PlanStatus = "running"
	PlanStatusSuccess   PlanStatus = "success"
	PlanStatusNoChanges PlanStatus = "no_changes"
	PlanStatusChanges   PlanStatus = "changes"
	PlanStatusFailed    PlanStatus = "failed"
)

// Valid reports whether the status is one of the supported plan outcomes.
func (s PlanStatus) Valid() bool {
	switch s {
	case PlanStatusPending, PlanStatusRunning, PlanStatusSuccess,
		PlanStatusNoChanges, PlanStatusChanges, PlanStatusFailed:
		return true
	default:
		return false
	}
}

// PlanStatusFromPlan determines the CI plan status from a parsed plan.
func PlanStatusFromPlan(hasChanges bool) PlanStatus {
	if !hasChanges {
		return PlanStatusNoChanges
	}
	return PlanStatusChanges
}

// PlanResult is the canonical representation of a single module's plan
// outcome — both for in-memory rendering and on-disk persistence.
//
//nolint:recvcheck // MarshalJSON/read APIs are value-based; UnmarshalJSON must mutate the receiver.
type PlanResult struct {
	moduleID          string
	modulePath        string
	components        map[string]string
	status            PlanStatus
	summary           string
	structuredDetails string
	rawPlanOutput     string
	errorMessage      string
	exitCode          int
	duration          time.Duration
}

// PlanResultOptions describes one module's Terraform plan outcome.
type PlanResultOptions struct {
	ModuleID          string
	ModulePath        string
	Components        map[string]string
	Status            PlanStatus
	Summary           string
	StructuredDetails string
	RawPlanOutput     string
	Error             string
	ExitCode          int
	Duration          time.Duration
}

type planResultJSON struct {
	ModuleID          string            `json:"module_id"`
	ModulePath        string            `json:"module_path"`
	Components        map[string]string `json:"components,omitempty"`
	Status            PlanStatus        `json:"status"`
	Summary           string            `json:"summary"`
	StructuredDetails string            `json:"structured_details,omitempty"`
	RawPlanOutput     string            `json:"raw_plan_output,omitempty"`
	Error             string            `json:"error,omitempty"`
	ExitCode          int               `json:"exit_code,omitempty"`
	Duration          time.Duration     `json:"duration,omitempty"`
}

// NewPlanResult validates and returns a plan result value.
func NewPlanResult(opts PlanResultOptions) (PlanResult, error) {
	if strings.TrimSpace(opts.ModuleID) == "" {
		return PlanResult{}, errPlanResultModuleIDRequired
	}
	if strings.TrimSpace(opts.ModulePath) == "" {
		return PlanResult{}, errPlanResultModulePathRequired
	}
	if !opts.Status.Valid() {
		return PlanResult{}, errInvalidPlanStatus(opts.Status)
	}
	return PlanResult{
		moduleID:          opts.ModuleID,
		modulePath:        opts.ModulePath,
		components:        cloneStringMap(opts.Components),
		status:            opts.Status,
		summary:           opts.Summary,
		structuredDetails: opts.StructuredDetails,
		rawPlanOutput:     opts.RawPlanOutput,
		errorMessage:      opts.Error,
		exitCode:          opts.ExitCode,
		duration:          opts.Duration,
	}, nil
}

// Clone returns a defensive copy of r.
func (r PlanResult) Clone() PlanResult {
	cloned := r
	cloned.components = cloneStringMap(r.components)
	return cloned
}

// ModuleID returns the stable module identifier.
func (r PlanResult) ModuleID() string { return r.moduleID }

// ModulePath returns the module path used for plan artifact discovery.
func (r PlanResult) ModulePath() string { return r.modulePath }

// Components returns a defensive copy of parsed module path components.
func (r PlanResult) Components() map[string]string { return cloneStringMap(r.components) }

// Status returns the module plan status.
func (r PlanResult) Status() PlanStatus { return r.status }

// Summary returns the short plan summary.
func (r PlanResult) Summary() string { return r.summary }

// StructuredDetails returns structured rendered plan details, when available.
func (r PlanResult) StructuredDetails() string { return r.structuredDetails }

// RawPlanOutput returns filtered raw plan text, when available.
func (r PlanResult) RawPlanOutput() string { return r.rawPlanOutput }

// ErrorMessage returns the plan result error message, if any.
func (r PlanResult) ErrorMessage() string { return r.errorMessage }

// ExitCode returns the Terraform plan exit code.
func (r PlanResult) ExitCode() int { return r.exitCode }

// Duration returns the plan command duration.
func (r PlanResult) Duration() time.Duration { return r.duration }

// Component returns the value of a named component from the Components map.
func (r PlanResult) Component(name string) string {
	if r.components != nil {
		return r.components[name]
	}
	return ""
}

// MarshalJSON preserves the public plan result artifact shape.
func (r PlanResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(planResultJSON{
		ModuleID:          r.moduleID,
		ModulePath:        r.modulePath,
		Components:        cloneStringMap(r.components),
		Status:            r.status,
		Summary:           r.summary,
		StructuredDetails: r.structuredDetails,
		RawPlanOutput:     r.rawPlanOutput,
		Error:             r.errorMessage,
		ExitCode:          r.exitCode,
		Duration:          r.duration,
	})
}

// UnmarshalJSON decodes the public plan result artifact shape.
func (r *PlanResult) UnmarshalJSON(data []byte) error {
	var raw planResultJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	result, err := NewPlanResult(PlanResultOptions(raw))
	if err != nil {
		return err
	}
	*r = result
	return nil
}

type planResultCollectionJSON struct {
	Results     []PlanResult `json:"results"`
	PipelineID  string       `json:"pipeline_id,omitempty"`
	CommitSHA   string       `json:"commit_sha,omitempty"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// PlanResultCollection is the persisted CI artifact collection for a pipeline run.
type PlanResultCollection struct {
	results     []PlanResult
	pipelineID  string
	commitSHA   string
	generatedAt time.Time
}

// PlanResultCollectionOptions describes a plan result collection.
type PlanResultCollectionOptions struct {
	Results     []PlanResult
	PipelineID  string
	CommitSHA   string
	GeneratedAt time.Time
}

// NewPlanResultCollection validates and returns an immutable plan result collection.
func NewPlanResultCollection(opts PlanResultCollectionOptions) (*PlanResultCollection, error) {
	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	results := make([]PlanResult, len(opts.Results))
	for i := range opts.Results {
		if strings.TrimSpace(opts.Results[i].ModuleID()) == "" {
			return nil, errPlanResultModuleIDRequired
		}
		if strings.TrimSpace(opts.Results[i].ModulePath()) == "" {
			return nil, errPlanResultModulePathRequired
		}
		if !opts.Results[i].Status().Valid() {
			return nil, errInvalidPlanStatus(opts.Results[i].Status())
		}
		results[i] = opts.Results[i].Clone()
	}
	return &PlanResultCollection{
		results:     results,
		pipelineID:  opts.PipelineID,
		commitSHA:   opts.CommitSHA,
		generatedAt: generatedAt.UTC(),
	}, nil
}

// EmptyPlanResultCollection returns an empty plan result collection.
func EmptyPlanResultCollection() *PlanResultCollection {
	return &PlanResultCollection{
		results:     []PlanResult{},
		generatedAt: time.Now().UTC(),
	}
}

// Clone returns a defensive copy of c.
func (c *PlanResultCollection) Clone() *PlanResultCollection {
	if c == nil {
		return nil
	}
	cloned, err := NewPlanResultCollection(PlanResultCollectionOptions{
		Results:     c.Results(),
		PipelineID:  c.pipelineID,
		CommitSHA:   c.commitSHA,
		GeneratedAt: c.generatedAt,
	})
	if err != nil {
		return nil
	}
	return cloned
}

// Results returns defensive copies of module plan results.
func (c *PlanResultCollection) Results() []PlanResult {
	if c == nil || len(c.results) == 0 {
		return nil
	}
	out := make([]PlanResult, len(c.results))
	for i := range c.results {
		out[i] = c.results[i].Clone()
	}
	return out
}

// PipelineID returns the collection CI pipeline ID, if known.
func (c *PlanResultCollection) PipelineID() string {
	if c == nil {
		return ""
	}
	return c.pipelineID
}

// CommitSHA returns the collection commit SHA, if known.
func (c *PlanResultCollection) CommitSHA() string {
	if c == nil {
		return ""
	}
	return c.commitSHA
}

// GeneratedAt returns the collection generation timestamp.
func (c *PlanResultCollection) GeneratedAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.generatedAt
}

// Len returns the number of plan results.
func (c *PlanResultCollection) Len() int {
	if c == nil {
		return 0
	}
	return len(c.results)
}

// Fingerprint returns a stable content fingerprint for the collection.
func (c *PlanResultCollection) Fingerprint() string {
	if c == nil {
		return ""
	}

	results := c.Results()
	sort.Slice(results, func(i, j int) bool {
		if results[i].ModuleID() == results[j].ModuleID() {
			return results[i].ModulePath() < results[j].ModulePath()
		}
		return results[i].ModuleID() < results[j].ModuleID()
	})

	payload := struct {
		Results []PlanResult `json:"results"`
	}{Results: results}

	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// MarshalJSON preserves the public plan result collection artifact shape.
func (c *PlanResultCollection) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("null"), nil
	}
	results := c.Results()
	if results == nil {
		results = []PlanResult{}
	}
	return json.Marshal(planResultCollectionJSON{
		Results:     results,
		PipelineID:  c.pipelineID,
		CommitSHA:   c.commitSHA,
		GeneratedAt: c.generatedAt,
	})
}

// UnmarshalJSON decodes the public plan result collection artifact shape.
func (c *PlanResultCollection) UnmarshalJSON(data []byte) error {
	var raw planResultCollectionJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	collection, err := NewPlanResultCollection(PlanResultCollectionOptions(raw))
	if err != nil {
		return err
	}
	*c = *collection
	return nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}

var (
	errPlanResultModuleIDRequired   = planResultError("plan result module_id is required")
	errPlanResultModulePathRequired = planResultError("plan result module_path is required")
)

type planResultError string

func (e planResultError) Error() string { return string(e) }

func errInvalidPlanStatus(status PlanStatus) error {
	return planResultError("plan result status " + string(status) + " is invalid")
}
