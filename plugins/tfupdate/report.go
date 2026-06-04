package tfupdate

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
)

type updateReportRequest struct {
	Result *tfupdateengine.UpdateResult
	Run    ci.ArtifactRun
}

func buildUpdateReport(req updateReportRequest) (*ci.Report, error) {
	result := req.Result
	status := ci.ReportStatusPass
	if result.Summary.Errors > 0 || result.Summary.UpdatesAvailable > 0 {
		status = ci.ReportStatusWarn
	}

	providerRows := make([]ci.RenderRow, 0, len(result.Providers))
	for i := range result.Providers {
		update := &result.Providers[i]
		if update.Status == domain.StatusUpToDate {
			continue
		}
		providerRows = append(providerRows, ci.NewRenderRow(
			ci.RenderModulePath(update.ModulePath()),
			ci.RenderCode(update.ProviderSource()),
			renderDisplayValue(update.DisplayCurrent()),
			renderDisplayValue(update.DisplayLatest()),
			ci.RenderLabel(update.StatusLabel(), renderUpdateStatusTone(update.Status)),
		))
	}

	moduleRows := make([]ci.RenderRow, 0, len(result.Modules))
	for i := range result.Modules {
		update := &result.Modules[i]
		if update.Status == domain.StatusUpToDate {
			continue
		}
		moduleRows = append(moduleRows, ci.NewRenderRow(
			ci.RenderModulePath(update.ModulePath()),
			ci.RenderCode(update.Source()),
			renderDisplayValue(update.DisplayCurrent()),
			renderDisplayValue(update.DisplayLatest()),
			ci.RenderLabel(update.StatusLabel(), renderUpdateStatusTone(update.Status)),
		))
	}

	summaryText := fmt.Sprintf(
		"%d checked, %d updates available, %d applied, %d errors",
		result.Summary.TotalChecked,
		result.Summary.UpdatesAvailable,
		result.Summary.UpdatesApplied,
		result.Summary.Errors,
	)
	blocks := make([]ci.RenderBlock, 0, 2)
	if len(providerRows) > 0 {
		blocks = append(blocks, ci.NewTableBlock("Providers", updateReportProviderColumns(), providerRows))
	}
	if len(moduleRows) > 0 {
		blocks = append(blocks, ci.NewTableBlock("Modules", updateReportModuleColumns(), moduleRows))
	}
	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: pluginName,
		Title:    "Dependency Update Check",
		Status:   status,
		Summary:  summaryText,
		Artifact: req.Run.Artifact(),
		Sections: []ci.RenderedSectionOptions{{
			Title:   "Dependency Update Check",
			Summary: summaryText,
			Blocks:  blocks,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("build tfupdate report: %w", err)
	}

	return report, nil
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

func renderDisplayValue(v string) ci.RenderValue {
	if v == "" {
		return ci.RenderText("-")
	}
	return ci.RenderText(v)
}

func renderUpdateStatusTone(status domain.UpdateStatus) ci.RenderTone {
	switch status {
	case domain.StatusApplied:
		return ci.RenderToneSuccess
	case domain.StatusUpdateAvailable, domain.StatusSkipped:
		return ci.RenderToneWarning
	case domain.StatusError:
		return ci.RenderToneFailure
	case domain.StatusUpToDate:
		return ci.RenderToneSuccess
	default:
		return ci.RenderToneNeutral
	}
}

func updateReportProviderColumns() []ci.RenderColumn {
	return []ci.RenderColumn{
		ci.NewRenderColumn("Module"),
		ci.NewRenderColumn("Provider"),
		ci.NewRenderColumn("Current"),
		ci.NewRenderColumn("Latest"),
		ci.NewRenderColumn("Status"),
	}
}

func updateReportModuleColumns() []ci.RenderColumn {
	return []ci.RenderColumn{
		ci.NewRenderColumn("Module"),
		ci.NewRenderColumn("Source"),
		ci.NewRenderColumn("Current"),
		ci.NewRenderColumn("Latest"),
		ci.NewRenderColumn("Status"),
	}
}
