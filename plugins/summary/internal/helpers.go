package summaryengine

import "github.com/edelwud/terraci/pkg/ci"

// HasReportableChanges returns true if there are plan changes, failures,
// or policy violations that warrant posting a comment.
func HasReportableChanges(plans []ci.ModulePlan, policySummary *ci.PolicySummary) bool {
	for i := range plans {
		if plans[i].Status == ci.PlanStatusChanges || plans[i].Status == ci.PlanStatusFailed {
			return true
		}
	}
	if policySummary != nil && (policySummary.HasFailures() || policySummary.HasWarnings()) {
		return true
	}
	return false
}
