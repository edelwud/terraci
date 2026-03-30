package cost

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func buildCostReport(result *model.EstimateResult) *ci.Report {
	visible := visibleReportModules(result.Modules)
	modules := make([]ci.ModuleReport, 0, len(visible))
	status := ci.ReportStatusPass

	for i := range visible {
		module := visible[i]
		moduleReport := ci.ModuleReport{
			ModulePath: module.ModulePath,
			Error:      module.Error,
		}
		if module.Error == "" {
			moduleReport.CostBefore = module.BeforeCost
			moduleReport.CostAfter = module.AfterCost
			moduleReport.CostDiff = module.DiffCost
			moduleReport.HasCost = true
		} else {
			status = ci.ReportStatusWarn
		}
		modules = append(modules, moduleReport)
	}

	return &ci.Report{
		Plugin:  "cost",
		Title:   "Cost Estimation",
		Status:  status,
		Summary: fmt.Sprintf("%d modules, total: $%.2f/mo (diff: %+.2f)", len(visible), result.TotalAfter, result.TotalDiff),
		Body:    renderCostReportBody(result),
		Modules: modules,
	}
}

func renderCostReportBody(result *model.EstimateResult) string {
	var b strings.Builder
	b.WriteString("| Module | Before | After | Diff | Notes |\n")
	b.WriteString("|--------|--------|-------|------|-------|\n")

	visible := visibleReportModules(result.Modules)
	for i := range visible {
		module := &visible[i]
		before := fmt.Sprintf("$%.2f", module.BeforeCost)
		after := fmt.Sprintf("$%.2f", module.AfterCost)
		diff := fmt.Sprintf("%+.2f", module.DiffCost)
		notes := ""

		if module.Error != "" {
			before = "-"
			after = "-"
			diff = "-"
			notes = module.Error
		}

		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			module.ModulePath, before, after, diff, notes)
	}

	fmt.Fprintf(&b, "\n**Total:** $%.2f/mo (diff: %+.2f)\n", result.TotalAfter, result.TotalDiff)
	return b.String()
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
	return module.Error != "" || module.BeforeCost != 0 || module.AfterCost != 0 || module.DiffCost != 0
}
