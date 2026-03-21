package ci

// HasReportableChanges returns true if there are plan changes, failures,
// or policy violations that warrant posting a comment.
func HasReportableChanges(plans []ModulePlan, policySummary *PolicySummary) bool {
	for i := range plans {
		if plans[i].Status == PlanStatusChanges || plans[i].Status == PlanStatusFailed {
			return true
		}
	}
	if policySummary != nil && (policySummary.HasFailures() || policySummary.HasWarnings()) {
		return true
	}
	return false
}
