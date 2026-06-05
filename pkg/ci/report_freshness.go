package ci

import (
	"fmt"
	"sort"

	"github.com/edelwud/terraci/pkg/diagnostic"
)

// ReportFreshnessStatus describes whether a producer report belongs to the
// current plan-result collection.
type ReportFreshnessStatus string

const (
	ReportFreshnessCurrent  ReportFreshnessStatus = "current"
	ReportFreshnessDegraded ReportFreshnessStatus = "degraded"
	ReportFreshnessStale    ReportFreshnessStatus = "stale"
)

// ReportFreshness is the freshness decision for one report.
type ReportFreshness struct {
	status     ReportFreshnessStatus
	diagnostic diagnostic.Diagnostic
}

// ReportSelection is the canonical selected-report result for consumers.
type ReportSelection struct {
	reports     ReportCollection
	diagnostics diagnostic.List
}

// ReportSelectionOptions controls report freshness selection.
type ReportSelectionOptions struct {
	Consumer         string
	ExcludeProducers []string
}

// EmptyReportSelection returns an empty selected-report result.
func EmptyReportSelection() ReportSelection {
	return ReportSelection{}
}

// SelectCurrentReports selects reports safe to render for the supplied plan
// collection. Reports with missing provenance or fingerprints are considered
// degraded but renderable; reports with mismatched non-empty fingerprints are
// skipped and returned as warnings.
func SelectCurrentReports(collection *PlanResultCollection, reports ReportCollection, opts ReportSelectionOptions) ReportSelection {
	excluded := make(map[string]struct{}, len(opts.ExcludeProducers))
	for _, producer := range opts.ExcludeProducers {
		excluded[producer] = struct{}{}
	}

	byProducer := make(map[string]*Report)
	diagnostics := diagnostic.List{}
	for _, report := range reports.Reports() {
		if report == nil {
			continue
		}
		if _, skip := excluded[report.Producer()]; skip {
			continue
		}
		freshness := EvaluateReportFreshness(collection, report, opts.Consumer)
		if freshness.Diagnostic().Valid() {
			diagnostics = diagnostics.Append(freshness.Diagnostic())
		}
		if freshness.Status() == ReportFreshnessStale {
			continue
		}
		byProducer[report.Producer()] = report
	}

	producers := make([]string, 0, len(byProducer))
	for producer := range byProducer {
		producers = append(producers, producer)
	}
	sort.Strings(producers)

	selectedReports := make([]*Report, 0, len(producers))
	for _, producer := range producers {
		selectedReports = append(selectedReports, byProducer[producer])
	}
	return ReportSelection{
		reports:     NewReportCollection(selectedReports...),
		diagnostics: diagnostics,
	}
}

// EvaluateReportFreshness evaluates one report against the current plan
// collection.
func EvaluateReportFreshness(collection *PlanResultCollection, report *Report, consumer string) ReportFreshness {
	if report == nil || report.Provenance() == nil {
		return ReportFreshness{status: ReportFreshnessDegraded}
	}

	reportFingerprint := report.Provenance().PlanResultsFingerprint()
	currentFingerprint := ""
	if collection != nil {
		currentFingerprint = collection.Fingerprint()
	}
	if reportFingerprint == "" || currentFingerprint == "" {
		return ReportFreshness{status: ReportFreshnessDegraded}
	}
	if reportFingerprint == currentFingerprint {
		return ReportFreshness{status: ReportFreshnessCurrent}
	}

	if consumer == "" {
		consumer = "ci"
	}
	return ReportFreshness{
		status: ReportFreshnessStale,
		diagnostic: diagnostic.Warning(fmt.Sprintf(
			"%s report %q skipped: plan_results_fingerprint %q does not match current %q",
			consumer,
			report.Producer(),
			reportFingerprint,
			currentFingerprint,
		), diagnostic.WithSource("report freshness")),
	}
}

// Status returns the freshness decision.
func (f ReportFreshness) Status() ReportFreshnessStatus { return f.status }

// Diagnostic returns the freshness diagnostic, if any.
func (f ReportFreshness) Diagnostic() diagnostic.Diagnostic { return f.diagnostic }

// ReportCollection returns selected reports in deterministic producer order.
func (s ReportSelection) ReportCollection() ReportCollection {
	return NewReportCollection(s.reports.Reports()...)
}

// Len returns the number of selected reports.
func (s ReportSelection) Len() int {
	return s.reports.Len()
}

// Producers returns selected report producer names in deterministic order.
func (s ReportSelection) Producers() []string {
	return s.reports.Producers()
}

// Diagnostics returns selection diagnostics.
func (s ReportSelection) Diagnostics() diagnostic.List { return s.diagnostics }
