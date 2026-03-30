package policy

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func buildPolicyReport(summary *policyengine.Summary) *ci.Report {
	status := ci.ReportStatusPass
	if summary.FailedModules > 0 {
		status = ci.ReportStatusFail
	} else if summary.WarnedModules > 0 {
		status = ci.ReportStatusWarn
	}

	return &ci.Report{
		Plugin:  "policy",
		Title:   "Policy Check",
		Status:  status,
		Summary: fmt.Sprintf("%d modules: %d passed, %d warned, %d failed", summary.TotalModules, summary.PassedModules, summary.WarnedModules, summary.FailedModules),
		Body:    renderPolicyReportBody(summary),
	}
}

func renderPolicyReportBody(summary *policyengine.Summary) string {
	var b strings.Builder
	for i := range summary.Results {
		result := &summary.Results[i]
		if result.Status() == "pass" {
			continue
		}
		fmt.Fprintf(&b, "**%s** (%s)\n", result.Module, result.Status())
		for _, failure := range result.Failures {
			fmt.Fprintf(&b, "- :x: %s", failure.Message)
			if failure.Namespace != "" {
				fmt.Fprintf(&b, " (%s)", failure.Namespace)
			}
			b.WriteString("\n")
		}
		for _, warning := range result.Warnings {
			fmt.Fprintf(&b, "- :warning: %s", warning.Message)
			if warning.Namespace != "" {
				fmt.Fprintf(&b, " (%s)", warning.Namespace)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}
