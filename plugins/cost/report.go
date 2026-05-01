package cost

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

const costChangesSectionKind ci.ReportSectionKind = "cost_changes"

type costChangesPayload struct {
	Totals costTotals      `json:"totals"`
	Rows   []costChangeRow `json:"rows,omitempty"`
}

type costTotals struct {
	Currency       string  `json:"currency,omitempty"`
	Before         float64 `json:"before,omitempty"`
	After          float64 `json:"after,omitempty"`
	Diff           float64 `json:"diff,omitempty"`
	UsageEstimated int     `json:"usage_estimated,omitempty"`
	UsageUnknown   int     `json:"usage_unknown,omitempty"`
	Unsupported    int     `json:"unsupported,omitempty"`
}

type costChangeRow struct {
	ModulePath string  `json:"module_path"`
	Before     float64 `json:"before,omitempty"`
	After      float64 `json:"after,omitempty"`
	Diff       float64 `json:"diff,omitempty"`
	HasCost    bool    `json:"has_cost,omitempty"`
	Error      string  `json:"error,omitempty"`
	Notes      string  `json:"notes,omitempty"`
}

func buildCostReport(result *model.EstimateResult) *ci.Report {
	visible := visibleReportModules(result.Modules)
	rows := make([]costChangeRow, 0, len(visible))
	status := ci.ReportStatusPass

	for i := range visible {
		module := visible[i]
		row := costChangeRow{
			ModulePath: module.ModulePath,
			Error:      module.Error,
		}
		if module.Error == "" {
			row.Before = module.BeforeCost
			row.After = module.AfterCost
			row.Diff = module.DiffCost
			row.HasCost = true
		} else {
			status = ci.ReportStatusWarn
		}
		rows = append(rows, row)
	}
	if len(result.PrefetchWarnings) > 0 {
		status = ci.ReportStatusWarn
	}
	if result.UsageUnknown > 0 || result.Unsupported > 0 {
		status = ci.ReportStatusWarn
	}

	payload, err := json.Marshal(costChangesPayload{
		Totals: costTotals{
			Currency:       result.Currency,
			Before:         result.TotalBefore,
			After:          result.TotalAfter,
			Diff:           result.TotalDiff,
			UsageEstimated: result.UsageEstimated,
			UsageUnknown:   result.UsageUnknown,
			Unsupported:    result.Unsupported,
		},
		Rows: rows,
	})
	if err != nil {
		panic(err)
	}

	return &ci.Report{
		Producer: pluginName,
		Title:    "Cost Estimation",
		Status:   status,
		Summary:  buildCostReportSummary(result, len(visible)),
		Sections: []ci.ReportSection{{
			Kind:           costChangesSectionKind,
			Title:          "Cost Estimation",
			Status:         status,
			SectionSummary: buildCostReportSummary(result, len(visible)),
			Payload:        payload,
		}},
	}
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

// saveArtifacts persists the estimation result and CI report to the service directory.
// Returns a joined error if one or both saves fail.
func saveArtifacts(serviceDir string, result *model.EstimateResult) error {
	if serviceDir == "" {
		return nil
	}

	var errs []error
	if err := ci.SaveJSON(serviceDir, resultsFile, result); err != nil {
		errs = append(errs, fmt.Errorf("save results: %w", err))
	}
	report := buildCostReport(result)
	if err := ci.SaveReport(serviceDir, report); err != nil {
		errs = append(errs, fmt.Errorf("save report: %w", err))
	}
	return errors.Join(errs...)
}
