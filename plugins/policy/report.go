package policy

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func buildPolicyReport(summary *policyengine.Summary) (*ci.Report, error) {
	status := ci.StatusFromCounts(summary.FailedModules, summary.WarnedModules)

	rows := make([]ci.FindingRow, 0, len(summary.Results))
	for i := range summary.Results {
		result := &summary.Results[i]
		row := ci.FindingRow{
			ModulePath: result.Module,
			Status:     ci.FindingRowStatusPass,
		}
		for _, failure := range result.Failures {
			row.Status = ci.FindingRowStatusFail
			row.Findings = append(row.Findings, ci.Finding{
				Severity:  ci.FindingSeverityFail,
				Message:   failure.Message,
				Namespace: failure.Namespace,
			})
		}
		for _, warning := range result.Warnings {
			if row.Status != ci.FindingRowStatusFail {
				row.Status = ci.FindingRowStatusWarn
			}
			row.Findings = append(row.Findings, ci.Finding{
				Severity:  ci.FindingSeverityWarn,
				Message:   warning.Message,
				Namespace: warning.Namespace,
			})
		}
		if len(row.Findings) > 0 {
			rows = append(rows, row)
		}
	}

	summaryText := fmt.Sprintf("%d modules: %d passed, %d warned, %d failed",
		summary.TotalModules, summary.PassedModules, summary.WarnedModules, summary.FailedModules)
	section, err := ci.EncodeSection(
		ci.ReportSectionKindFindings,
		"Policy Check",
		summaryText,
		status,
		ci.FindingsSection{Rows: rows},
	)
	if err != nil {
		return nil, fmt.Errorf("build policy report: %w", err)
	}

	return ci.BuildReport("policy", "Policy Check", status, summaryText, section), nil
}
