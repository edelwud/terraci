package cost

import (
	"context"
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

type costReportRequest struct {
	Result *model.EstimateResult
	Run    ci.ArtifactRun
}

func buildCostReport(req costReportRequest) (*ci.Report, error) {
	result := req.Result
	visible := visibleReportModules(result.Modules)
	rows := make([]ci.RenderRow, 0, len(visible))
	errorItems := make([]ci.RenderValue, 0)
	status := ci.ReportStatusPass

	for i := range visible {
		module := visible[i]
		if module.Error != "" {
			errorItems = append(errorItems, formatModuleError(costReportModuleLabel(module), module.Error))
			status = ci.ReportStatusWarn
			continue
		}
		rows = append(rows, ci.NewRenderRow(
			ci.RenderModulePath(costReportModuleLabel(module)),
			ci.RenderMoney(module.BeforeCost, monthlyMoney()),
			ci.RenderMoney(module.AfterCost, monthlyMoney()),
			ci.RenderMoneyDelta(module.DiffCost, monthlyMoney()),
		))
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
		blocks = append(blocks, ci.NewTableBlock("", []ci.RenderColumn{
			ci.NewRenderColumn("Module"),
			ci.NewRenderColumn("Before"),
			ci.NewRenderColumn("After"),
			ci.NewRenderColumn("Diff"),
		}, rows))
	}
	if len(errorItems) > 0 {
		blocks = append(blocks, ci.NewListBlock("Estimation errors", errorItems))
	}
	if limitations := buildCostLimitations(result); len(limitations) > 0 {
		blocks = append(blocks, ci.NewListBlock("Limitations", limitations))
	}
	blocks = append(blocks, buildCostTotalBlock(result))
	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: pluginName,
		Title:    costReportTitle,
		Status:   status,
		Summary:  summary,
		Artifact: req.Run.Artifact(),
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
		fmt.Sprintf("%d modules", moduleCount),
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

func formatModuleError(module, err string) ci.RenderValue {
	if strings.TrimSpace(module) == "" {
		return ci.RenderText(err)
	}
	return ci.RenderInline(ci.RenderModulePath(module), ci.RenderText(": "+err))
}

func buildCostLimitations(result *model.EstimateResult) []ci.RenderValue {
	if result == nil {
		return nil
	}
	items := make([]ci.RenderValue, 0)
	if result.UsageEstimated > 0 {
		items = append(items, ci.RenderText(fmt.Sprintf("%d usage-based resources estimated from plan-time assumptions", result.UsageEstimated)))
	}
	if result.UsageUnknown > 0 {
		items = append(items, ci.RenderText(fmt.Sprintf("%d usage-based resources missing usage inputs", result.UsageUnknown)))
	}
	if result.Unsupported > 0 {
		items = append(items, ci.RenderText(fmt.Sprintf("%d unsupported resources omitted from totals", result.Unsupported)))
	}
	for i := range result.PrefetchWarnings {
		items = append(items, formatPrefetchWarning(result.PrefetchWarnings[i]))
	}
	return items
}

func formatPrefetchWarning(w model.PrefetchDiagnostic) ci.RenderValue {
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
		return ci.RenderText(w.Kind)
	}
	if w.Kind == "" {
		return ci.RenderText(strings.Join(parts, ": "))
	}
	return ci.RenderText(fmt.Sprintf("%s: %s", w.Kind, strings.Join(parts, ": ")))
}

func buildCostTotalBlock(result *model.EstimateResult) ci.RenderBlock {
	if result == nil {
		return ci.NewTextBlock(ci.RenderText("Total: "), ci.RenderMoney(0, monthlyMoney()))
	}
	if model.CostIsZero(result.TotalDiff) {
		return ci.NewTextBlock(ci.RenderText("Total: "), ci.RenderMoney(result.TotalAfter, monthlyMoney()))
	}
	return ci.NewTextBlock(
		ci.RenderText("Total: "),
		ci.RenderMoney(result.TotalBefore, monthlyMoney()),
		ci.RenderText(" -> "),
		ci.RenderMoney(result.TotalAfter, monthlyMoney()),
		ci.RenderText(" ("),
		ci.RenderMoneyDelta(result.TotalDiff, monthlyMoney()),
		ci.RenderText(")"),
	)
}

func monthlyMoney() ci.RenderMoneyOptions {
	return ci.RenderMoneyOptions{Unit: ci.RenderMoneyUnitMonth}
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

// saveArtifacts persists the estimation result and CI report to the service directory.
// Returns a joined error if one or both saves fail.
func saveArtifacts(ctx context.Context, appCtx *plugin.AppContext, result *model.EstimateResult, collection *ci.PlanResultCollection) error {
	if appCtx == nil || appCtx.Reports() == nil {
		return nil
	}
	publication, err := ci.NewArtifactPublication(ci.ArtifactPublicationOptions{
		Producer: pluginName,
		Results:  ci.RawResults(result),
		BuildReport: func() (*ci.Report, error) {
			run, err := plugin.NewArtifactRun(appCtx, plugin.ArtifactRunOptions{
				Producer:   pluginName,
				Collection: collection,
			})
			if err != nil {
				return nil, fmt.Errorf("artifact run: %w", err)
			}
			return buildCostReport(costReportRequest{Result: result, Run: run})
		},
	})
	if err != nil {
		return err
	}
	return appCtx.Reports().PublishArtifacts(ctx, publication)
}
