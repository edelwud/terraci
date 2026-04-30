package ci

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

// ModulePlan represents terraform plan output for a single module
type ModulePlan struct {
	ModuleID          string
	ModulePath        string
	Components        map[string]string
	Status            PlanStatus
	Summary           string // Compact summary e.g., "+2 ~1 -1"
	StructuredDetails string // Structured resource list by action (markdown)
	RawPlanOutput     string // Filtered raw plan output (diff only)
	Error             string // Error message if plan failed
	Duration          time.Duration
	// Cost estimation fields
	CostBefore float64 // Monthly cost before changes (USD)
	CostAfter  float64 // Monthly cost after changes (USD)
	CostDiff   float64 // Cost difference (after - before)
	HasCost    bool    // True if cost was calculated
}

// Get returns the value of a named component from the Components map.
func (p *ModulePlan) Get(name string) string {
	if p.Components != nil {
		return p.Components[name]
	}
	return ""
}

// PlanStatus represents the status of a terraform plan
type PlanStatus string

const (
	PlanStatusPending   PlanStatus = "pending"
	PlanStatusRunning   PlanStatus = "running"
	PlanStatusSuccess   PlanStatus = "success"
	PlanStatusNoChanges PlanStatus = "no_changes"
	PlanStatusChanges   PlanStatus = "changes"
	PlanStatusFailed    PlanStatus = "failed"
)

// PlanStatusFromPlan determines the CI plan status from a parsed plan.
func PlanStatusFromPlan(hasChanges bool) PlanStatus {
	if !hasChanges {
		return PlanStatusNoChanges
	}
	return PlanStatusChanges
}

// PlanResult represents a persisted CI artifact for a single module plan.
type PlanResult struct {
	ModuleID          string            `json:"module_id"`
	ModulePath        string            `json:"module_path"`
	Components        map[string]string `json:"components,omitempty"`
	Status            PlanStatus        `json:"status"`
	Summary           string            `json:"summary"`
	StructuredDetails string            `json:"structured_details,omitempty"`
	RawPlanOutput     string            `json:"raw_plan_output,omitempty"`
	Error             string            `json:"error,omitempty"`
	ExitCode          int               `json:"exit_code"`
	// Cost estimation fields
	CostBefore float64 `json:"cost_before,omitempty"`
	CostAfter  float64 `json:"cost_after,omitempty"`
	CostDiff   float64 `json:"cost_diff,omitempty"`
	HasCost    bool    `json:"has_cost,omitempty"`
}

// Get returns the value of a named component from the Components map.
func (r *PlanResult) Get(name string) string {
	if r.Components != nil {
		return r.Components[name]
	}
	return ""
}

// PlanResultCollection is the persisted CI artifact collection for a pipeline run.
type PlanResultCollection struct {
	Results     []PlanResult `json:"results"`
	PipelineID  string       `json:"pipeline_id,omitempty"`
	CommitSHA   string       `json:"commit_sha,omitempty"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// Fingerprint returns a stable content fingerprint for the collection.
func (c *PlanResultCollection) Fingerprint() string {
	if c == nil {
		return ""
	}

	type fingerprintResult PlanResult

	results := make([]fingerprintResult, 0, len(c.Results))
	for i := range c.Results {
		results = append(results, fingerprintResult(c.Results[i]))
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].ModuleID == results[j].ModuleID {
			return results[i].ModulePath < results[j].ModulePath
		}
		return results[i].ModuleID < results[j].ModuleID
	})

	payload := struct {
		Results []fingerprintResult `json:"results"`
	}{
		Results: results,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ToModulePlans converts plan results to ModulePlan for comment rendering
func (c *PlanResultCollection) ToModulePlans() []ModulePlan {
	plans := make([]ModulePlan, len(c.Results))
	for i := range c.Results {
		r := &c.Results[i]
		plans[i] = ModulePlan{
			ModuleID:          r.ModuleID,
			ModulePath:        r.ModulePath,
			Components:        r.Components,
			Status:            r.Status,
			Summary:           r.Summary,
			StructuredDetails: r.StructuredDetails,
			RawPlanOutput:     r.RawPlanOutput,
			Error:             r.Error,
			CostBefore:        r.CostBefore,
			CostAfter:         r.CostAfter,
			CostDiff:          r.CostDiff,
			HasCost:           r.HasCost,
		}
	}
	return plans
}
