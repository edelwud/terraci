package tfupdate

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
)

func buildUpdateReport(result *tfupdateengine.UpdateResult) (*ci.Report, error) {
	status := ci.ReportStatusPass
	if result.Summary.Errors > 0 || result.Summary.UpdatesAvailable > 0 {
		status = ci.ReportStatusWarn
	}

	rows := make([]ci.DependencyUpdateRow, 0, len(result.Providers)+len(result.Modules))
	for i := range result.Providers {
		update := &result.Providers[i]
		rows = append(rows, ci.DependencyUpdateRow{
			ModulePath: update.ModulePath(),
			Kind:       ci.DependencyKindProvider,
			Name:       update.ProviderSource(),
			Current:    update.DisplayCurrent(),
			Latest:     update.DisplayLatest(),
			Bumped:     update.DisplayAvailable(),
			Status:     mapUpdateStatus(update.Status),
			Issue:      update.Issue,
		})
	}

	for i := range result.Modules {
		update := &result.Modules[i]
		rows = append(rows, ci.DependencyUpdateRow{
			ModulePath: update.ModulePath(),
			Kind:       ci.DependencyKindModule,
			Name:       update.Source(),
			Current:    update.DisplayCurrent(),
			Latest:     update.DisplayLatest(),
			Bumped:     update.DisplayAvailable(),
			Status:     mapUpdateStatus(update.Status),
			Issue:      update.Issue,
		})
	}

	summaryText := fmt.Sprintf(
		"%d checked, %d updates available, %d applied, %d errors",
		result.Summary.TotalChecked,
		result.Summary.UpdatesAvailable,
		result.Summary.UpdatesApplied,
		result.Summary.Errors,
	)
	section, err := ci.EncodeSection(
		ci.ReportSectionKindDependencyUpdates,
		"Dependency Update Check",
		summaryText,
		status,
		ci.DependencyUpdatesSection{Rows: rows},
	)
	if err != nil {
		return nil, fmt.Errorf("build tfupdate report: %w", err)
	}

	return &ci.Report{
		Producer:   "tfupdate",
		Title:      "Dependency Update Check",
		Status:     status,
		Summary:    summaryText,
		Provenance: ci.NewProvenance("", "", ""),
		Sections:   []ci.ReportSection{section},
	}, nil
}

func mapUpdateStatus(status domain.UpdateStatus) ci.DependencyUpdateStatus {
	switch status {
	case domain.StatusUpdateAvailable:
		return ci.DependencyUpdateStatusUpdateAvailable
	case domain.StatusApplied:
		return ci.DependencyUpdateStatusApplied
	case domain.StatusSkipped:
		return ci.DependencyUpdateStatusSkipped
	case domain.StatusError:
		return ci.DependencyUpdateStatusError
	case domain.StatusUpToDate:
		return ci.DependencyUpdateStatusUpToDate
	default:
		return ci.DependencyUpdateStatusUpToDate
	}
}

func renderReportBody(result *tfupdateengine.UpdateResult) string {
	var b strings.Builder

	if len(result.Providers) > 0 {
		b.WriteString("### Providers\n\n")
		b.WriteString("| Module | Provider | Current | Latest | Status |\n")
		b.WriteString("|--------|----------|---------|--------|--------|\n")
		for i := range result.Providers {
			update := &result.Providers[i]
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				update.ModulePath(), update.ProviderSource(), update.DisplayCurrent(), update.DisplayLatest(), update.StatusLabel())
		}
		b.WriteString("\n")
	}

	if len(result.Modules) > 0 {
		b.WriteString("### Modules\n\n")
		b.WriteString("| Module | Source | Current | Latest | Status |\n")
		b.WriteString("|--------|--------|---------|--------|--------|\n")
		for i := range result.Modules {
			update := &result.Modules[i]
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				update.ModulePath(), update.Source(), update.DisplayCurrent(), update.DisplayLatest(), update.StatusLabel())
		}
	}

	return b.String()
}
