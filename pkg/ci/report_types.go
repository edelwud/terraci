package ci

import "time"

// Report is a plugin-produced CI enrichment artifact consumed by summary flows.
// Plugins write reports as {serviceDir}/{plugin}-report.json.
type Report struct {
	Plugin     string            `json:"plugin"`
	Title      string            `json:"title"`
	Status     ReportStatus      `json:"status"`
	Summary    string            `json:"summary"`
	Provenance *ReportProvenance `json:"provenance,omitempty"`
	Sections   []ReportSection   `json:"sections,omitempty"`
}

// ReportProvenance captures the source run identity for a persisted report.
type ReportProvenance struct {
	Producer               string    `json:"producer,omitempty"`
	GeneratedAt            time.Time `json:"generated_at"`
	CommitSHA              string    `json:"commit_sha,omitempty"`
	PipelineID             string    `json:"pipeline_id,omitempty"`
	PlanResultsFingerprint string    `json:"plan_results_fingerprint,omitempty"`
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

// Valid reports whether the status is one of the supported CI report outcomes.
func (s ReportStatus) Valid() bool {
	switch s {
	case ReportStatusPass, ReportStatusWarn, ReportStatusFail:
		return true
	default:
		return false
	}
}
