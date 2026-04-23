package ci

import (
	"encoding/json"
	"fmt"
	"time"
)

// CommentMarker is used to identify TerraCI review comments for updates.
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

// Report is a plugin-produced CI enrichment artifact consumed by summary flows.
// Plugins write reports as {serviceDir}/{plugin}-report.json.
type Report struct {
	Plugin   string          `json:"plugin"`
	Title    string          `json:"title"`
	Status   ReportStatus    `json:"status"`
	Summary  string          `json:"summary"`
	Sections []ReportSection `json:"sections,omitempty"`
}

// ReportSectionKind identifies the shape of one typed report section.
type ReportSectionKind string

const (
	ReportSectionKindOverview          ReportSectionKind = "overview"
	ReportSectionKindModuleTable       ReportSectionKind = "module_table"
	ReportSectionKindFindings          ReportSectionKind = "findings"
	ReportSectionKindDependencyUpdates ReportSectionKind = "dependency_updates"
	ReportSectionKindCostChanges       ReportSectionKind = "cost_changes"
)

// ReportSection is a discriminated union over supported typed report payloads.
type ReportSection struct {
	Kind              ReportSectionKind         `json:"kind"`
	Title             string                    `json:"title,omitempty"`
	Status            ReportStatus              `json:"status,omitempty"`
	SectionSummary    string                    `json:"section_summary,omitempty"`
	Overview          *OverviewSection          `json:"overview,omitempty"`
	ModuleTable       *ModuleTableSection       `json:"module_table,omitempty"`
	Findings          *FindingsSection          `json:"findings,omitempty"`
	DependencyUpdates *DependencyUpdatesSection `json:"dependency_updates,omitempty"`
	CostChanges       *CostChangesSection       `json:"cost_changes,omitempty"`
}

type reportSectionPayload struct {
	name    string
	present bool
}

// ModuleTableSection groups module-oriented rows such as terraform plan results.
type ModuleTableSection struct {
	Environment string           `json:"environment,omitempty"`
	Rows        []ModuleTableRow `json:"rows,omitempty"`
}

// ModuleTableRow is one actionable module entry in a module table section.
type ModuleTableRow struct {
	ModuleID          string     `json:"module_id"`
	ModulePath        string     `json:"module_path"`
	Status            PlanStatus `json:"status"`
	Summary           string     `json:"summary,omitempty"`
	Error             string     `json:"error,omitempty"`
	StructuredDetails string     `json:"structured_details,omitempty"`
	RawPlanOutput     string     `json:"raw_plan_output,omitempty"`
	CostBefore        float64    `json:"cost_before,omitempty"`
	CostAfter         float64    `json:"cost_after,omitempty"`
	CostDiff          float64    `json:"cost_diff,omitempty"`
	HasCost           bool       `json:"has_cost,omitempty"`
}

// CostChangesSection holds structured cost estimation data.
type CostChangesSection struct {
	Totals CostTotals      `json:"totals"`
	Rows   []CostChangeRow `json:"rows,omitempty"`
}

// CostTotals are aggregate cost values for one report.
type CostTotals struct {
	Currency       string  `json:"currency,omitempty"`
	Before         float64 `json:"before,omitempty"`
	After          float64 `json:"after,omitempty"`
	Diff           float64 `json:"diff,omitempty"`
	UsageEstimated int     `json:"usage_estimated,omitempty"`
	UsageUnknown   int     `json:"usage_unknown,omitempty"`
	Unsupported    int     `json:"unsupported,omitempty"`
}

// CostChangeRow is one actionable module-level cost result.
type CostChangeRow struct {
	ModulePath string  `json:"module_path"`
	Before     float64 `json:"before,omitempty"`
	After      float64 `json:"after,omitempty"`
	Diff       float64 `json:"diff,omitempty"`
	HasCost    bool    `json:"has_cost,omitempty"`
	Error      string  `json:"error,omitempty"`
	Notes      string  `json:"notes,omitempty"`
}

// FindingsSection holds warned/failed findings for modules or resources.
type FindingsSection struct {
	Rows []FindingRow `json:"rows,omitempty"`
}

// FindingRow is one module or resource result in a findings report.
type FindingRow struct {
	ModulePath string           `json:"module_path"`
	Status     FindingRowStatus `json:"status"`
	Findings   []Finding        `json:"findings,omitempty"`
}

// FindingRowStatus is the outcome for one findings row.
type FindingRowStatus string

const (
	FindingRowStatusPass FindingRowStatus = "pass"
	FindingRowStatusWarn FindingRowStatus = "warn"
	FindingRowStatusFail FindingRowStatus = "fail"
)

// Finding describes one warned/failed issue.
type Finding struct {
	Severity  FindingSeverity `json:"severity"`
	Message   string          `json:"message"`
	Namespace string          `json:"namespace,omitempty"`
}

// FindingSeverity is the severity of a finding.
type FindingSeverity string

const (
	FindingSeverityWarn FindingSeverity = "warn"
	FindingSeverityFail FindingSeverity = "fail"
)

// DependencyUpdatesSection holds actionable dependency update rows.
type DependencyUpdatesSection struct {
	Rows []DependencyUpdateRow `json:"rows,omitempty"`
}

// DependencyUpdateStatus is the outcome of a dependency update check.
type DependencyUpdateStatus string

const (
	DependencyUpdateStatusUpToDate        DependencyUpdateStatus = "up_to_date"
	DependencyUpdateStatusUpdateAvailable DependencyUpdateStatus = "update_available"
	DependencyUpdateStatusApplied         DependencyUpdateStatus = "applied"
	DependencyUpdateStatusSkipped         DependencyUpdateStatus = "skipped"
	DependencyUpdateStatusError           DependencyUpdateStatus = "error"
)

// DependencyKind identifies the dependency category for an update row.
type DependencyKind string

const (
	DependencyKindProvider DependencyKind = "provider"
	DependencyKindModule   DependencyKind = "module"
)

// DependencyUpdateRow is one dependency update result.
type DependencyUpdateRow struct {
	ModulePath string                 `json:"module_path"`
	Kind       DependencyKind         `json:"kind"`
	Name       string                 `json:"name"`
	Current    string                 `json:"current,omitempty"`
	Latest     string                 `json:"latest,omitempty"`
	Bumped     string                 `json:"bumped,omitempty"`
	Status     DependencyUpdateStatus `json:"status"`
	Issue      string                 `json:"issue,omitempty"`
}

// OverviewSection is the aggregate summary report payload.
type OverviewSection struct {
	PlanStats SummaryPlanStats        `json:"plan_stats"`
	Reports   []SummaryReportOverview `json:"reports,omitempty"`
}

// SummaryPlanStats tracks aggregate terraform plan counts.
type SummaryPlanStats struct {
	Total     int `json:"total"`
	Success   int `json:"success,omitempty"`
	NoChanges int `json:"no_changes,omitempty"`
	Changes   int `json:"changes,omitempty"`
	Failed    int `json:"failed,omitempty"`
	Pending   int `json:"pending,omitempty"`
	Running   int `json:"running,omitempty"`
}

// SummaryReportOverview captures one contributing report at a glance.
type SummaryReportOverview struct {
	Kind    ReportSectionKind `json:"kind"`
	Title   string            `json:"title"`
	Status  ReportStatus      `json:"status"`
	Summary string            `json:"summary,omitempty"`
}

// ReportStatus indicates the outcome of a plugin's check.
type ReportStatus string

const (
	ReportStatusPass ReportStatus = "pass"
	ReportStatusWarn ReportStatus = "warn"
	ReportStatusFail ReportStatus = "fail"
)

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

// MarshalJSON validates the section union before encoding.
func (s ReportSection) MarshalJSON() ([]byte, error) {
	if err := validateReportSection(s); err != nil {
		return nil, err
	}

	type alias ReportSection
	return json.Marshal(alias(s))
}

// UnmarshalJSON decodes and validates the section union.
func (s *ReportSection) UnmarshalJSON(data []byte) error {
	type alias ReportSection
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	section := ReportSection(decoded)
	if err := validateReportSection(section); err != nil {
		return err
	}

	*s = section
	return nil
}

func validateReportSection(s ReportSection) error {
	payloads := []reportSectionPayload{
		{name: "overview", present: s.Overview != nil},
		{name: "module_table", present: s.ModuleTable != nil},
		{name: "findings", present: s.Findings != nil},
		{name: "dependency_updates", present: s.DependencyUpdates != nil},
		{name: "cost_changes", present: s.CostChanges != nil},
	}

	expectedPayload, err := expectedReportSectionPayload(s.Kind)
	if err != nil {
		return err
	}

	matched := 0
	for i := range payloads {
		payload := payloads[i]
		if !payload.present {
			continue
		}
		matched++
		if payload.name != expectedPayload {
			return fmt.Errorf("report section %q has unexpected %s payload", s.Kind, payload.name)
		}
	}

	if matched == 0 {
		return fmt.Errorf("report section %q is missing %s payload", s.Kind, expectedPayload)
	}
	if matched != 1 {
		return fmt.Errorf("report section %q must contain exactly one payload", s.Kind)
	}

	return nil
}

func expectedReportSectionPayload(kind ReportSectionKind) (string, error) {
	switch kind {
	case ReportSectionKindOverview:
		return "overview", nil
	case ReportSectionKindModuleTable:
		return "module_table", nil
	case ReportSectionKindFindings:
		return "findings", nil
	case ReportSectionKindDependencyUpdates:
		return "dependency_updates", nil
	case ReportSectionKindCostChanges:
		return "cost_changes", nil
	default:
		return "", fmt.Errorf("unknown report section kind %q", kind)
	}
}
