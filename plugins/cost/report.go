package cost

import (
	"errors"
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func buildCostReport(result *model.EstimateResult) (*ci.Report, error) {
	visible := visibleReportModules(result.Modules)
	rows := make([][]string, 0, len(visible))
	status := ci.ReportStatusPass

	for i := range visible {
		module := visible[i]
		before := "-"
		after := "-"
		diff := "-"
		notes := "-"
		if module.Error == "" {
			before = reportMonthlyCost(module.BeforeCost)
			after = reportMonthlyCost(module.AfterCost)
			diff = reportCostDiff(module.DiffCost)
		} else {
			notes = module.Error
			status = ci.ReportStatusWarn
		}
		rows = append(rows, []string{module.ModulePath, before, after, diff, notes})
	}
	if len(result.PrefetchWarnings) > 0 {
		status = ci.ReportStatusWarn
	}
	if result.UsageUnknown > 0 || result.Unsupported > 0 {
		status = ci.ReportStatusWarn
	}

	summary := buildCostReportSummary(result, len(visible))
	blocks := make([]ci.RenderBlock, 0, 2)
	if len(rows) > 0 {
		blocks = append(blocks, ci.RenderTableBlock("", []string{"Module", "Before", "After", "Diff", "Notes"}, rows))
	}
	blocks = append(blocks, ci.RenderTextBlock(fmt.Sprintf(
		"Totals: %s %s -> %s",
		reportMonthlyCost(result.TotalBefore),
		reportCostDiff(result.TotalDiff),
		reportMonthlyCost(result.TotalAfter),
	)))
	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: pluginName,
		Title:    costReportTitle,
		Status:   status,
		Summary:  summary,
		Sections: []ci.RenderedSectionOptions{{
			Title:   costReportTitle,
			Summary: summary,
			Blocks:  blocks,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("build cost report: %w", err)
	}

	return report, nil
}

func buildCostReportSummary(result *model.EstimateResult, moduleCount int) string {
	parts := []string{
		fmt.Sprintf("%d modules, total: $%.2f/mo (diff: %+.2f)", moduleCount, result.TotalAfter, result.TotalDiff),
	}
	if result.UsageEstimated > 0 {
		parts = append(parts, fmt.Sprintf("usage estimated: %d", result.UsageEstimated))
	}
	if result.UsageUnknown > 0 {
		parts = append(parts, fmt.Sprintf("usage unknown: %d", result.UsageUnknown))
	}
	if result.Unsupported > 0 {
		parts = append(parts, fmt.Sprintf("unsupported: %d", result.Unsupported))
	}
	return strings.Join(parts, "; ")
}

func visibleReportModules(modules []model.ModuleCost) []model.ModuleCost {
	visible := make([]model.ModuleCost, 0, len(modules))
	for i := range modules {
		if shouldShowReportModule(&modules[i]) {
			visible = append(visible, modules[i])
		}
	}
	return visible
}

func shouldShowReportModule(module *model.ModuleCost) bool {
	if module == nil {
		return false
	}
	return module.Error != "" || !model.CostIsZero(module.BeforeCost) || !model.CostIsZero(module.AfterCost) || !model.CostIsZero(module.DiffCost)
}

func reportMonthlyCost(cost float64) string {
	if cost == 0 {
		return "$0"
	}
	if cost < 0.01 {
		return "<$0.01"
	}
	if cost >= 1000 {
		return fmt.Sprintf("$%.0f", cost)
	}
	if cost >= 1 {
		return fmt.Sprintf("$%.2f", cost)
	}
	return fmt.Sprintf("$%.4f", cost)
}

func reportCostDiff(diff float64) string {
	if diff == 0 {
		return "$0"
	}
	if diff > 0 {
		if diff >= 1000 {
			return fmt.Sprintf("+$%.0f", diff)
		}
		if diff >= 1 {
			return fmt.Sprintf("+$%.2f", diff)
		}
		return fmt.Sprintf("+$%.4f", diff)
	}
	diff = -diff
	if diff >= 1000 {
		return fmt.Sprintf("-$%.0f", diff)
	}
	if diff >= 1 {
		return fmt.Sprintf("-$%.2f", diff)
	}
	return fmt.Sprintf("-$%.4f", diff)
}

// saveArtifacts persists the estimation result and CI report to the service directory.
// Returns a joined error if one or both saves fail.
func saveArtifacts(serviceDir string, result *model.EstimateResult) error {
	report, err := buildCostReport(result)
	if err != nil {
		// Still attempt to save the raw results so the user can inspect them.
		return errors.Join(err, ci.SaveResultsAndReport(serviceDir, resultsFile, result, nil))
	}
	return ci.SaveResultsAndReport(serviceDir, resultsFile, result, report)
}
