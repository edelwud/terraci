package summaryengine

import (
	"context"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/plugins/internal/diagnosticlog"
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
	Snapshot          SummarySnapshot
	Body              string
	Labels            []string
	LabelDiagnostics  diagnostic.List
	ReportDiagnostics diagnostic.List
	PostedComment     bool
	SyncedLabels      bool
	SkippedReason     string
	ProviderDetected  bool
}

// Run scans plans, loads reports, renders a summary comment, posts it, and
// optionally synchronizes TerraCI-managed PR/MR labels.
func Run(ctx context.Context, runtime Runtime, _ Request) (*Result, error) {
	log.Info("scanning for plan results")
	collection, err := loadPlanResults(runtime)
	if err != nil {
		return nil, err
	}

	result := &Result{Snapshot: NewSummarySnapshot(SummarySnapshotOptions{PlanResults: collection})}
	if collection == nil || collection.Len() == 0 {
		result.SkippedReason = "no_plan_results"
		log.Warn("no plan results found, skipping summary")
		return result, nil
	}

	log.WithField("count", collection.Len()).Info("found plan results")

	selection, err := loadReportSelection(ctx, runtime, collection)
	if err != nil {
		return nil, err
	}
	result.Snapshot = NewSummarySnapshot(SummarySnapshotOptions{
		PlanResults: collection,
		Reports:     selection.ReportCollection(),
	})
	result.ReportDiagnostics = selection.Diagnostics()
	diagnosticlog.Log(result.ReportDiagnostics)

	if runtime.Config.OnChangesOnly && !result.Snapshot.HasReportableChanges() {
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

	labelResult := resolveSummaryLabels(runtime, result.Snapshot.PlanResults())
	result.Labels = labelResult.Labels
	result.LabelDiagnostics = labelResult.Diagnostics
	diagnosticlog.Log(labelResult.Diagnostics)

	body, err := composeSummaryBody(runtime, result.Snapshot, provider, result.Labels)
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

func resolveSummaryLabels(runtime Runtime, collection *ci.PlanResultCollection) LabelResult {
	return ResolveLabels(LabelRequest{
		WorkDir:     runtime.WorkDir,
		Segments:    runtime.Segments,
		PlanResults: collection,
		Templates:   runtime.Config.Labels,
		Parser:      runtime.LabelParser,
	})
}
