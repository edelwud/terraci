package ci

import (
	"fmt"
	"sort"
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
	Status  ReportFreshnessStatus
	Warning string
}

// ReportSelection is the canonical selected-report result for consumers.
type ReportSelection struct {
	Reports  []*Report
	Warnings []string
}

// ReportSelectionOptions controls report freshness selection.
type ReportSelectionOptions struct {
	Consumer         string
	ExcludeProducers []string
}

// SelectCurrentReports selects reports safe to render for the supplied plan
// collection. Reports with missing provenance or fingerprints are considered
// degraded but renderable; reports with mismatched non-empty fingerprints are
// skipped and returned as warnings.
func SelectCurrentReports(collection *PlanResultCollection, reports []*Report, opts ReportSelectionOptions) ReportSelection {
	excluded := make(map[string]struct{}, len(opts.ExcludeProducers))
	for _, producer := range opts.ExcludeProducers {
		excluded[producer] = struct{}{}
	}

	byProducer := make(map[string]*Report)
	warningSet := make(map[string]struct{})
	for _, report := range reports {
		if report == nil {
			continue
		}
		if _, skip := excluded[report.Producer]; skip {
			continue
		}
		freshness := EvaluateReportFreshness(collection, report, opts.Consumer)
		if freshness.Warning != "" {
			warningSet[freshness.Warning] = struct{}{}
		}
		if freshness.Status == ReportFreshnessStale {
			continue
		}
		byProducer[report.Producer] = report
	}

	producers := make([]string, 0, len(byProducer))
	for producer := range byProducer {
		producers = append(producers, producer)
	}
	sort.Strings(producers)

	warnings := make([]string, 0, len(warningSet))
	for warning := range warningSet {
		warnings = append(warnings, warning)
	}
	sort.Strings(warnings)

	selected := ReportSelection{
		Reports:  make([]*Report, 0, len(producers)),
		Warnings: warnings,
	}
	for _, producer := range producers {
		selected.Reports = append(selected.Reports, byProducer[producer].Clone())
	}
	return selected
}

// EvaluateReportFreshness evaluates one report against the current plan
// collection.
func EvaluateReportFreshness(collection *PlanResultCollection, report *Report, consumer string) ReportFreshness {
	if report == nil || report.Provenance == nil {
		return ReportFreshness{Status: ReportFreshnessDegraded}
	}

	reportFingerprint := report.Provenance.PlanResultsFingerprint
	currentFingerprint := ""
	if collection != nil {
		currentFingerprint = collection.Fingerprint()
	}
	if reportFingerprint == "" || currentFingerprint == "" {
		return ReportFreshness{Status: ReportFreshnessDegraded}
	}
	if reportFingerprint == currentFingerprint {
		return ReportFreshness{Status: ReportFreshnessCurrent}
	}

	if consumer == "" {
		consumer = "ci"
	}
	return ReportFreshness{
		Status: ReportFreshnessStale,
		Warning: fmt.Sprintf(
			"%s report %q skipped: plan_results_fingerprint %q does not match current %q",
			consumer,
			report.Producer,
			reportFingerprint,
			currentFingerprint,
		),
	}
}
