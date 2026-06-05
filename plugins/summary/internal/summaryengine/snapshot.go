package summaryengine

import "github.com/edelwud/terraci/pkg/ci"

// SummarySnapshot is the immutable summary composition input for plan results
// and selected plugin reports.
type SummarySnapshot struct {
	planResults *ci.PlanResultCollection
	reports     ci.ReportCollection
}

// SummarySnapshotOptions describes a summary composition snapshot.
type SummarySnapshotOptions struct {
	PlanResults *ci.PlanResultCollection
	Reports     ci.ReportCollection
}

// NewSummarySnapshot builds a defensive summary composition snapshot.
func NewSummarySnapshot(opts SummarySnapshotOptions) SummarySnapshot {
	planResults := opts.PlanResults.Clone()
	if planResults == nil {
		planResults = ci.EmptyPlanResultCollection()
	}
	return SummarySnapshot{
		planResults: planResults,
		reports:     ci.NewReportCollection(opts.Reports.Reports()...),
	}
}

// PlanResults returns the plan result collection for this snapshot.
func (s SummarySnapshot) PlanResults() *ci.PlanResultCollection {
	if s.planResults == nil {
		return ci.EmptyPlanResultCollection()
	}
	return s.planResults.Clone()
}

// Reports returns the selected plugin reports for this snapshot.
func (s SummarySnapshot) Reports() ci.ReportCollection {
	return ci.NewReportCollection(s.reports.Reports()...)
}

// Plans returns defensive plan result values for internal iteration.
func (s SummarySnapshot) Plans() []ci.PlanResult {
	if s.planResults == nil {
		return nil
	}
	return s.planResults.Results()
}

// HasReportableChanges reports whether the snapshot has module or report signal
// worth posting.
func (s SummarySnapshot) HasReportableChanges() bool {
	plans := s.Plans()
	for i := range plans {
		if plans[i].Status() == ci.PlanStatusChanges || plans[i].Status() == ci.PlanStatusFailed {
			return true
		}
	}
	for _, report := range s.reports.Reports() {
		if report.Status() == ci.ReportStatusWarn || report.Status() == ci.ReportStatusFail {
			return true
		}
	}
	return false
}
