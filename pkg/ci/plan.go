package ci

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
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

// PlanStatusFromPlan determines the CI plan status from a parsed plan.
func PlanStatusFromPlan(hasChanges bool) PlanStatus {
	if !hasChanges {
		return PlanStatusNoChanges
	}
	return PlanStatusChanges
}

// PlanResult is the canonical representation of a single module's plan
// outcome — both for in-memory rendering and on-disk persistence.
type PlanResult struct {
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

	results := append([]PlanResult(nil), c.Results...)
	sort.Slice(results, func(i, j int) bool {
		if results[i].ModuleID == results[j].ModuleID {
			return results[i].ModulePath < results[j].ModulePath
		}
		return results[i].ModuleID < results[j].ModuleID
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
