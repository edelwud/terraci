package tfupdate

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
)

func buildUpdateReport(result *tfupdateengine.UpdateResult) *ci.Report {
	status := ci.ReportStatusPass
	if result.Summary.Errors > 0 || result.Summary.UpdatesAvailable > 0 {
		status = ci.ReportStatusWarn
	}

	return &ci.Report{
		Plugin: "tfupdate",
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
