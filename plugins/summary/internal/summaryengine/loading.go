package summaryengine

import (
	"fmt"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/planresults"
)

// PlanScanner loads plan result artifacts.
type PlanScanner interface {
	ScanPlanResults(rootDir string, segments []string) (*ci.PlanResultCollection, error)
}

// ReportLoader loads producer reports from the service directory.
type ReportLoader interface {
	LoadReports(serviceDir string) ([]*ci.Report, error)
}

type reportSelection struct {
	reports  []*ci.Report
	warnings []string
}

type defaultPlanScanner struct{}

func (defaultPlanScanner) ScanPlanResults(rootDir string, segments []string) (*ci.PlanResultCollection, error) {
	return planresults.Scan(rootDir, segments)
}

type defaultReportLoader struct{}

func (defaultReportLoader) LoadReports(serviceDir string) ([]*ci.Report, error) {
	return ci.LoadReports(serviceDir)
}

func loadPlanResults(runtime Runtime) (*ci.PlanResultCollection, error) {
	collection, err := planScanner(runtime).ScanPlanResults(runtime.WorkDir, runtime.Segments)
	if err != nil {
		return nil, fmt.Errorf("failed to scan plan results: %w", err)
	}
	return collection, nil
}

func loadReportSelection(runtime Runtime, collection *ci.PlanResultCollection) (reportSelection, error) {
	reports, err := reportLoader(runtime).LoadReports(runtime.ServiceDir)
	if err != nil {
		return reportSelection{}, fmt.Errorf("failed to load plugin reports: %w", err)
	}
	return selectCurrentReports(collection, filterSummaryReports(reports)), nil
}

func selectCurrentReports(collection *ci.PlanResultCollection, reports []*ci.Report) reportSelection {
	fingerprint := collection.Fingerprint()
	selected := reportSelection{reports: make([]*ci.Report, 0, len(reports))}
	for _, report := range reports {
		skip, warning := reportProvenanceWarning(report, fingerprint)
		if warning != "" {
			selected.warnings = append(selected.warnings, warning)
		}
		if skip {
			continue
		}
		selected.reports = append(selected.reports, report)
	}
	return selected
}

func reportProvenanceWarning(report *ci.Report, currentFingerprint string) (skip bool, warning string) {
	if report == nil {
		return false, ""
	}
	if report.Provenance == nil {
		return false, ""
	}
	reportFingerprint := report.Provenance.PlanResultsFingerprint
	if reportFingerprint == "" {
		return false, ""
	}
	if currentFingerprint == "" {
		return false, ""
	}
	if reportFingerprint != currentFingerprint {
		return true, fmt.Sprintf("summary report %q skipped: plan_results_fingerprint %q does not match current %q", report.Producer, reportFingerprint, currentFingerprint)
	}
	return false, ""
}

func planScanner(runtime Runtime) PlanScanner {
	if runtime.PlanScanner != nil {
		return runtime.PlanScanner
	}
	return defaultPlanScanner{}
}

func reportLoader(runtime Runtime) ReportLoader {
	if runtime.ReportLoader != nil {
		return runtime.ReportLoader
	}
	return defaultReportLoader{}
}

func logWarnings(warnings []string) {
	for _, warning := range warnings {
		log.Warn(warning)
	}
}
