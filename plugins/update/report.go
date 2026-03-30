package update

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

func buildUpdateReport(result *updateengine.UpdateResult) *ci.Report {
	status := ci.ReportStatusPass
	if result.Summary.Errors > 0 || result.Summary.UpdatesAvailable > 0 {
		status = ci.ReportStatusWarn
	}

	return &ci.Report{
		Plugin: "update",
		Title:  "Dependency Update Check",
		Status: status,
		Summary: fmt.Sprintf(
			"%d checked, %d updates available, %d applied, %d errors",
			result.Summary.TotalChecked,
			result.Summary.UpdatesAvailable,
			result.Summary.UpdatesApplied,
			result.Summary.Errors,
		),
		Body: renderReportBody(result),
	}
}

func renderReportBody(result *updateengine.UpdateResult) string {
	var b strings.Builder

	if len(result.Providers) > 0 {
		b.WriteString("### Providers\n\n")
		b.WriteString("| Module | Provider | Current | Latest | Status |\n")
		b.WriteString("|--------|----------|---------|--------|--------|\n")
		for i := range result.Providers {
			update := &result.Providers[i]
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				update.ModulePath, update.ProviderSource, reportCurrent(update.Constraint, update.CurrentVersion), update.LatestVersion, providerReportStatus(update))
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
				update.ModulePath, update.Source, reportCurrent(update.Constraint, update.CurrentVersion), update.LatestVersion, moduleReportStatus(update))
		}
	}

	return b.String()
}

func reportCurrent(constraint, current string) string {
	if current != "" {
		return current
	}
	return constraint
}

func providerReportStatus(update *updateengine.ProviderVersionUpdate) string {
	switch {
	case update.Skipped:
		return "skipped: " + update.SkipReason
	case update.Error != "":
		return "error: " + update.Error
	case update.Applied:
		return "applied"
	case update.UpdateAvailable:
		return "update available"
	default:
		return "up to date"
	}
}

func moduleReportStatus(update *updateengine.ModuleVersionUpdate) string {
	switch {
	case update.Skipped:
		return "skipped: " + update.SkipReason
	case update.Error != "":
		return "error: " + update.Error
	case update.Applied:
		return "applied"
	case update.UpdateAvailable:
		return "update available"
	default:
		return "up to date"
	}
}
