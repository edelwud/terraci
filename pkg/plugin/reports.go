package plugin

import (
	"sync"

	"github.com/edelwud/terraci/pkg/ci"
)

// ReportRegistry allows plugins to publish and consume reports in-memory.
// In CI (multi-process), reports are still written to JSON files for artifacts.
// The registry provides an in-process fast path for single-process runs.
type ReportRegistry struct {
	mu      sync.RWMutex
	reports map[string]*ci.Report
}

// NewReportRegistry creates a new empty ReportRegistry.
func NewReportRegistry() *ReportRegistry {
	return &ReportRegistry{
		reports: make(map[string]*ci.Report),
	}
}

// Publish stores a report in the registry, keyed by plugin name.
func (r *ReportRegistry) Publish(report *ci.Report) {
	if report == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports[report.Plugin] = cloneReport(report)
}

// Get retrieves a report by plugin name.
func (r *ReportRegistry) Get(pluginName string) (*ci.Report, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rep, ok := r.reports[pluginName]
	if !ok {
		return nil, false
	}
	return cloneReport(rep), true
}

// All returns all published reports.
func (r *ReportRegistry) All() []*ci.Report {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ci.Report, 0, len(r.reports))
	for _, rep := range r.reports {
		result = append(result, cloneReport(rep))
	}
	return result
}

func cloneReport(report *ci.Report) *ci.Report {
	if report == nil {
		return nil
	}

	cloned := *report
	if report.Provenance != nil {
		provenance := *report.Provenance
		cloned.Provenance = &provenance
	}
	cloned.Sections = cloneReportSections(report.Sections)
	return &cloned
}

func cloneReportSections(sections []ci.ReportSection) []ci.ReportSection {
	if len(sections) == 0 {
		return nil
	}

	cloned := make([]ci.ReportSection, len(sections))
	for i := range sections {
		cloned[i] = cloneReportSection(sections[i])
	}
	return cloned
}

func cloneReportSection(section ci.ReportSection) ci.ReportSection {
	cloned := section
	if section.Overview != nil {
		overview := *section.Overview
		overview.Reports = append([]ci.SummaryReportOverview(nil), section.Overview.Reports...)
		cloned.Overview = &overview
	}
	if section.ModuleTable != nil {
		moduleTable := *section.ModuleTable
		moduleTable.Rows = cloneModuleTableRows(section.ModuleTable.Rows)
		cloned.ModuleTable = &moduleTable
	}
	if section.Findings != nil {
		findings := *section.Findings
		findings.Rows = cloneFindingRows(section.Findings.Rows)
		cloned.Findings = &findings
	}
	if section.DependencyUpdates != nil {
		updates := *section.DependencyUpdates
		updates.Rows = append([]ci.DependencyUpdateRow(nil), section.DependencyUpdates.Rows...)
		cloned.DependencyUpdates = &updates
	}
	if section.CostChanges != nil {
		costChanges := *section.CostChanges
		costChanges.Rows = append([]ci.CostChangeRow(nil), section.CostChanges.Rows...)
		cloned.CostChanges = &costChanges
	}
	return cloned
}

func cloneModuleTableRows(rows []ci.ModuleTableRow) []ci.ModuleTableRow {
	return append([]ci.ModuleTableRow(nil), rows...)
}

func cloneFindingRows(rows []ci.FindingRow) []ci.FindingRow {
	if len(rows) == 0 {
		return nil
	}

	cloned := make([]ci.FindingRow, len(rows))
	for i := range rows {
		cloned[i] = rows[i]
		cloned[i].Findings = append([]ci.Finding(nil), rows[i].Findings...)
	}
	return cloned
}
