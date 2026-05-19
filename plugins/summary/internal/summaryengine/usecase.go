package summaryengine

import (
	"context"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
)

// ReportProducer is the producer name used by summary's own reports/comments.
const ReportProducer = "summary"

// Runtime holds normalized summary configuration and command-independent dependencies.
type Runtime struct {
	Config           Config
	WorkDir          string
	ServiceDir       string
	Segments         []string
	ProviderResolver ProviderResolver
	PlanScanner      PlanScanner
	ReportStore      ci.ReportStore
	LabelParser      PlanParser
}

// Request is reserved for command-time options. The summary command currently
// has no flags; keeping the request explicit keeps the use case stable.
type Request struct{}

// Result reports what the summary use case did.
type Result struct {
	Collection       *ci.PlanResultCollection
	Plans            []ci.PlanResult
	Reports          []*ci.Report
	Body             string
	Labels           []string
	LabelWarnings    []string
	ReportWarnings   []string
	PostedComment    bool
	SyncedLabels     bool
	SkippedReason    string
	ProviderDetected bool
}

// Run scans plans, loads reports, renders a summary comment, posts it, and
// optionally synchronizes TerraCI-managed PR/MR labels.
func Run(ctx context.Context, runtime Runtime, _ Request) (*Result, error) {
	log.Info("scanning for plan results")
	collection, err := loadPlanResults(runtime)
	if err != nil {
		return nil, err
	}

	result := &Result{Collection: collection}
	if collection == nil || len(collection.Results) == 0 {
		result.SkippedReason = "no_plan_results"
		log.Warn("no plan results found, skipping summary")
		return result, nil
	}

	log.WithField("count", len(collection.Results)).Info("found plan results")
	result.Plans = append([]ci.PlanResult(nil), collection.Results...)

	selection, err := loadReportSelection(ctx, runtime, collection)
	if err != nil {
		return nil, err
	}
	result.Reports = selection.Reports
	result.ReportWarnings = selection.Warnings
	logWarnings(result.ReportWarnings)

	if runtime.Config.OnChangesOnly && !HasReportableChanges(result.Plans, result.Reports) {
		result.SkippedReason = "no_reportable_changes"
		log.Info("no reportable changes, skipping comment")
		return result, nil
	}

	provider, resolveErr := resolveProvider(runtime.ProviderResolver)
	if resolveErr != nil || provider == nil {
		result.SkippedReason = "no_provider"
		log.Info("no CI provider detected, printing summary only")
		return result, nil //nolint:nilerr // no provider is intentionally non-fatal
	}
	result.ProviderDetected = true

	labelResult := resolveSummaryLabels(runtime, result.Plans)
	result.Labels = labelResult.Labels
	result.LabelWarnings = labelResult.Warnings
	logWarnings(labelResult.Warnings)

	body, err := composeSummaryBody(runtime, collection, result.Plans, result.Reports, provider, result.Labels)
	if err != nil {
		return result, err
	}
	result.Body = body

	commentSvc, ok := provider.CommentService()
	if !ok || !commentSvc.IsEnabled() {
		result.SkippedReason = "comment_disabled"
		log.Info("PR/MR comments disabled or no token available")
		return result, nil
	}

	if err := postSummary(ctx, summaryPostRequest{body: body, labels: result.Labels, commentSvc: commentSvc}, result); err != nil {
		return result, err
	}

	return result, nil
}

func resolveSummaryLabels(runtime Runtime, plans []ci.PlanResult) LabelResult {
	return ResolveLabels(LabelRequest{
		WorkDir:   runtime.WorkDir,
		Segments:  runtime.Segments,
		Plans:     plans,
		Templates: runtime.Config.Labels,
		Parser:    runtime.LabelParser,
	})
}

// HasReportableChanges reports whether the run has any module or report signal worth posting.
func HasReportableChanges(plans []ci.PlanResult, reports []*ci.Report) bool {
	for i := range plans {
		if plans[i].Status == ci.PlanStatusChanges || plans[i].Status == ci.PlanStatusFailed {
			return true
		}
	}
	for _, r := range reports {
		if r.Status == ci.ReportStatusWarn || r.Status == ci.ReportStatusFail {
			return true
		}
	}
	return false
}

func filterSummaryReports(reports []*ci.Report) []*ci.Report {
	filtered := make([]*ci.Report, 0, len(reports))
	for _, report := range reports {
		if report == nil || report.Producer == ReportProducer {
			continue
		}
		filtered = append(filtered, report)
	}
	return filtered
}
