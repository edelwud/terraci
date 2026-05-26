package cost

import (
	"context"
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/internal/reportctx"
)

type costReportRequest struct {
	Result *model.EstimateResult
	Run    ci.ArtifactRun
}

func buildCostReport(req costReportRequest) (*ci.Report, error) {
	result := req.Result
	visible := visibleReportModules(result.Modules)
	rows := make([][]string, 0, len(visible))
	errorItems := make([]string, 0)
	status := ci.ReportStatusPass

	for i := range visible {
		module := visible[i]
		if module.Error != "" {
			errorItems = append(errorItems, formatModuleError(costReportModuleLabel(module), module.Error))
			status = ci.ReportStatusWarn
			continue
		}
		rows = append(rows, []string{
			costReportModuleLabel(module),
			reportMonthlyCost(module.BeforeCost),
			reportMonthlyCost(module.AfterCost),
			reportCostDiff(module.DiffCost),
		})
	}
	for _, moduleError := range result.Errors {
		if moduleError.Error == "" {
			continue
		}
		errorItems = append(errorItems, formatModuleError(moduleError.ModuleID, moduleError.Error))
		status = ci.ReportStatusWarn
	}
	if len(result.PrefetchWarnings) > 0 {
		status = ci.ReportStatusWarn
	}
	if result.UsageUnknown > 0 || result.Unsupported > 0 {
		status = ci.ReportStatusWarn
	}

	summary := buildCostReportSummary(result, len(visible))
	blocks := make([]ci.RenderBlock, 0, 4)
	if len(rows) > 0 {
		blocks = append(blocks, ci.RenderTableBlock("", []string{"Module", "Before", "After", "Diff"}, rows))
	}
	if len(errorItems) > 0 {
		blocks = append(blocks, ci.RenderListBlock("Estimation errors", errorItems))
	}
	if limitations := buildCostLimitations(result); len(limitations) > 0 {
		blocks = append(blocks, ci.RenderListBlock("Limitations", limitations))
	}
	blocks = append(blocks, ci.RenderTextBlock(formatCostReportTotal(result)))
	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: pluginName,
		Title:    costReportTitle,
		Status:   status,
		Summary:  summary,
		Artifact: req.Run.Artifact,
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
		fmt.Sprintf("%d modules, %s", moduleCount, formatCostSummaryTotal(result)),
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

func costReportModuleLabel(module model.ModuleCost) string {
	if strings.TrimSpace(module.ModulePath) != "" {
		return module.ModulePath
	}
	return module.ModuleID
}

func formatModuleError(module, err string) string {
	if strings.TrimSpace(module) == "" {
		return err
	}
	return fmt.Sprintf("%s: %s", module, err)
}

func buildCostLimitations(result *model.EstimateResult) []string {
	if result == nil {
		return nil
	}
	items := make([]string, 0)
	if result.UsageEstimated > 0 {
		items = append(items, fmt.Sprintf("%d usage-based resources estimated from plan-time assumptions", result.UsageEstimated))
	}
	if result.UsageUnknown > 0 {
		items = append(items, fmt.Sprintf("%d usage-based resources missing usage inputs", result.UsageUnknown))
	}
	if result.Unsupported > 0 {
		items = append(items, fmt.Sprintf("%d unsupported resources omitted from totals", result.Unsupported))
	}
	for i := range result.PrefetchWarnings {
		items = append(items, formatPrefetchWarning(result.PrefetchWarnings[i]))
	}
	return items
}

func formatPrefetchWarning(w model.PrefetchDiagnostic) string {
	parts := make([]string, 0, 4)
	if w.ModuleID != "" {
		parts = append(parts, w.ModuleID)
	}
	if w.ResourceType != "" {
		parts = append(parts, w.ResourceType)
	}
	if w.Address != "" {
		parts = append(parts, w.Address)
	}
	if w.Detail != "" {
		parts = append(parts, w.Detail)
	}
	if len(parts) == 0 {
		return w.Kind
	}
	if w.Kind == "" {
		return strings.Join(parts, ": ")
	}
	return fmt.Sprintf("%s: %s", w.Kind, strings.Join(parts, ": "))
}

func formatCostSummaryTotal(result *model.EstimateResult) string {
	if result == nil {
		return "total: $0/mo"
	}
	total := fmt.Sprintf("total: %s/mo", reportMonthlyCost(result.TotalAfter))
	if model.CostIsZero(result.TotalDiff) {
		return total
	}
	return fmt.Sprintf("%s (diff: %s/mo)", total, reportCostDiff(result.TotalDiff))
}

func formatCostReportTotal(result *model.EstimateResult) string {
	if result == nil {
		return "Total: $0/mo"
	}
	after := reportMonthlyCost(result.TotalAfter) + "/mo"
	if model.CostIsZero(result.TotalDiff) {
		return "Total: " + after
	}
	before := reportMonthlyCost(result.TotalBefore) + "/mo"
	diff := reportCostDiff(result.TotalDiff) + "/mo"
	return fmt.Sprintf("Total: %s -> %s (%s)", before, after, diff)
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
	if model.CostIsZero(cost) {
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
	if model.CostIsZero(diff) {
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
func saveArtifacts(ctx context.Context, appCtx *plugin.AppContext, result *model.EstimateResult, collection *ci.PlanResultCollection) error {
	if appCtx == nil || appCtx.Reports() == nil {
		return nil
	}
	return ci.PublishArtifacts(ctx, ci.PublishArtifactsRequest{
		Producer: pluginName,
		Writer:   appCtx.Reports(),
		Results:  result,
		BuildReport: func() (*ci.Report, error) {
			run, err := reportctx.NewRun(appCtx, reportctx.Options{
				Producer:   pluginName,
				Collection: collection,
			})
			if err != nil {
				return nil, fmt.Errorf("artifact run: %w", err)
			}
			return buildCostReport(costReportRequest{Result: result, Run: run})
		},
	})
}
