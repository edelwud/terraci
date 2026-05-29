package summaryengine

import (
	"context"
	"fmt"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/planresults"
)

// PlanScanner loads plan result artifacts.
type PlanScanner interface {
	ScanPlanResults(rootDir string, segments []string) (*ci.PlanResultCollection, error)
}

type defaultPlanScanner struct{}

func (defaultPlanScanner) ScanPlanResults(rootDir string, segments []string) (*ci.PlanResultCollection, error) {
	return planresults.Scan(rootDir, segments)
}

func loadPlanResults(runtime Runtime) (*ci.PlanResultCollection, error) {
	collection, err := planScanner(runtime).ScanPlanResults(runtime.WorkDir, runtime.Segments)
	if err != nil {
		return nil, fmt.Errorf("failed to scan plan results: %w", err)
	}
	return collection, nil
}

func loadReportSelection(ctx context.Context, runtime Runtime, collection *ci.PlanResultCollection) (ci.ReportSelection, error) {
	reports, err := reportStore(runtime).LoadReports(ctx)
	if err != nil {
		return ci.ReportSelection{}, fmt.Errorf("failed to load plugin reports: %w", err)
	}
	return ci.SelectCurrentReports(collection, filterSummaryReports(reports), ci.ReportSelectionOptions{
		Consumer: "summary",
	}), nil
}

func planScanner(runtime Runtime) PlanScanner {
	if runtime.PlanScanner != nil {
		return runtime.PlanScanner
	}
	return defaultPlanScanner{}
}

func reportStore(runtime Runtime) ci.ReportStore {
	if runtime.ReportStore != nil {
		return runtime.ReportStore
	}
	if runtime.ServiceDir != "" {
		return ci.NewFileReportStore(runtime.ServiceDir)
	}
	return ci.NewMemoryReportStore()
}

func logDiagnostics(diags diagnostic.List) {
	for _, diag := range diags.All() {
		entry := log.WithField("severity", diag.Severity())
		if diag.Source() != "" {
			entry = entry.WithField("source", diag.Source())
		}
		if diag.Module() != "" {
			entry = entry.WithField("module", diag.Module())
		}
		if diag.Cause() != nil {
			entry = entry.WithError(diag.Cause())
		}
		switch diag.Severity() {
		case diagnostic.SeverityError:
			entry.Error(diag.Message())
		case diagnostic.SeverityWarning:
			entry.Warn(diag.Message())
		case diagnostic.SeverityInfo:
			entry.Info(diag.Message())
		default:
			entry.Info(diag.Message())
		}
	}
}
