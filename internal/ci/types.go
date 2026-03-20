package ci

import (
	"time"

	"github.com/edelwud/terraci/internal/policy"
)

// CommentMarker is used to identify terraci comments for updates
const CommentMarker = "<!-- terraci-plan-comment -->"

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

// CommentData contains all data needed to render a PR/MR comment
type CommentData struct {
	Plans         []ModulePlan
	PolicySummary *policy.Summary
	PipelineURL   string
	PipelineID    string
	CommitSHA     string
	GeneratedAt   time.Time
	TotalModules  int
}

// PlanResult represents the result of a terraform plan for a single module
type PlanResult struct {
	ModuleID          string            `json:"module_id"`
	ModulePath        string            `json:"module_path"`
	Service           string            `json:"service"`
	Environment       string            `json:"environment"`
	Region            string            `json:"region"`
	Module            string            `json:"module"`
	Submodule         string            `json:"submodule,omitempty"`
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

// Get returns the value of a named component, checking Components first,
// then falling back to fixed fields for backward compatibility with old JSON.
func (r *PlanResult) Get(name string) string {
	if r.Components != nil {
		if v, ok := r.Components[name]; ok {
			return v
		}
	}
	// Backward compat fallback for old JSON without Components
	switch name {
	case "service":
		return r.Service
	case "environment":
		return r.Environment
	case "region":
		return r.Region
	case "module":
		return r.Module
	case "submodule":
		return r.Submodule
	}
	return ""
}

// PlanResultCollection is a collection of plan results from multiple jobs
type PlanResultCollection struct {
	Results     []PlanResult `json:"results"`
	PipelineID  string       `json:"pipeline_id,omitempty"`
	CommitSHA   string       `json:"commit_sha,omitempty"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// ToModulePlans converts plan results to ModulePlan for comment rendering
func (c *PlanResultCollection) ToModulePlans() []ModulePlan {
	plans := make([]ModulePlan, len(c.Results))
	for i := range c.Results {
		r := &c.Results[i]
		components := r.Components
		if components == nil {
			// Backward compat: build from fixed fields
			components = map[string]string{}
			if r.Service != "" {
				components["service"] = r.Service
			}
			if r.Environment != "" {
				components["environment"] = r.Environment
			}
			if r.Region != "" {
				components["region"] = r.Region
			}
			if r.Module != "" {
				components["module"] = r.Module
			}
			if r.Submodule != "" {
				components["submodule"] = r.Submodule
			}
		}
		plans[i] = ModulePlan{
			ModuleID:          r.ModuleID,
			ModulePath:        r.ModulePath,
			Components:        components,
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
