package summaryengine

import (
	"context"
	"fmt"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/planresults"
)

// ReportProducer is the producer name used by summary's own reports/comments.
const ReportProducer = "summary"

// Provider is the CI-provider surface needed by the summary use case.
type Provider interface {
	CommitSHA() string
	PipelineID() string
	CommentService() (ci.CommentService, bool)
}

// ProviderResolver resolves the active CI provider for the current command.
type ProviderResolver func() (Provider, error)

// PlanScanner loads plan result artifacts.
type PlanScanner interface {
	ScanPlanResults(rootDir string, segments []string) (*ci.PlanResultCollection, error)
}

// ReportLoader loads producer reports from the service directory.
type ReportLoader interface {
	LoadReports(serviceDir string) ([]*ci.Report, error)
}

// Runtime holds normalized summary configuration and command-independent dependencies.
type Runtime struct {
	Config           Config
	WorkDir          string
	ServiceDir       string
	Segments         []string
	ProviderResolver ProviderResolver
	PlanScanner      PlanScanner
	ReportLoader     ReportLoader
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
	PostedComment    bool
	SyncedLabels     bool
	SkippedReason    string
	ProviderDetected bool
}

type defaultPlanScanner struct{}

func (defaultPlanScanner) ScanPlanResults(rootDir string, segments []string) (*ci.PlanResultCollection, error) {
	return planresults.Scan(rootDir, segments)
}

type defaultReportLoader struct{}

func (defaultReportLoader) LoadReports(serviceDir string) ([]*ci.Report, error) {
	return ci.LoadReports(serviceDir)
}

// Run scans plans, loads reports, renders a summary comment, posts it, and
// optionally synchronizes TerraCI-managed PR/MR labels.
func Run(ctx context.Context, runtime Runtime, _ Request) (*Result, error) {
	log.Info("scanning for plan results")
	collection, err := planScanner(runtime).ScanPlanResults(runtime.WorkDir, runtime.Segments)
	if err != nil {
		return nil, fmt.Errorf("failed to scan plan results: %w", err)
	}

	result := &Result{Collection: collection}
	if collection == nil || len(collection.Results) == 0 {
		result.SkippedReason = "no_plan_results"
		log.Warn("no plan results found, skipping summary")
		return result, nil
	}

	log.WithField("count", len(collection.Results)).Info("found plan results")
	result.Plans = append([]ci.PlanResult(nil), collection.Results...)

	reports, err := reportLoader(runtime).LoadReports(runtime.ServiceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin reports: %w", err)
	}
	result.Reports = filterSummaryReports(reports)

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
	logLabelWarnings(labelResult.Warnings)

	body, err := ComposeCommentWithOptions(
		result.Plans,
		result.Reports,
		provider.CommitSHA(),
		provider.PipelineID(),
		collection.GeneratedAt,
		runtime.Config.IncludeDetailsEnabled(),
	)
	if err != nil {
		return result, fmt.Errorf("compose summary comment: %w", err)
	}
	body = ci.EmbedManagedLabels(body, result.Labels)
	result.Body = body

	commentSvc, ok := provider.CommentService()
	if !ok || !commentSvc.IsEnabled() {
		result.SkippedReason = "comment_disabled"
		log.Info("PR/MR comments disabled or no token available")
		return result, nil
	}

	var previousLabels []string
	var managedSvc ci.ManagedLabelService
	if svc, ok := commentSvc.(ci.ManagedLabelService); ok {
		managedSvc = svc
		currentBody, found, err := svc.CurrentCommentBody(ctx)
		if err != nil {
			log.Warnf("failed to read current TerraCI comment for managed labels: %v", err)
		} else if found {
			previousLabels = ci.ExtractManagedLabels(currentBody)
		}
	}

	log.Info("updating PR/MR comment")
	if err := commentSvc.UpsertComment(ctx, body); err != nil {
		return result, fmt.Errorf("failed to update comment: %w", err)
	}
	result.PostedComment = true
	log.Info("comment updated successfully")

	if managedSvc != nil && (len(previousLabels) > 0 || len(result.Labels) > 0) {
		if err := managedSvc.SyncLabels(ctx, previousLabels, result.Labels); err != nil {
			log.Warnf("failed to synchronize summary labels: %v", err)
		} else {
			result.SyncedLabels = true
		}
	}

	return result, nil
}

func planScanner(runtime Runtime) PlanScanner {
	if runtime.PlanScanner != nil {
		return runtime.PlanScanner
	}
	return defaultPlanScanner{}
}

func reportLoader(runtime Runtime) ReportLoader {
	if runtime.ReportLoader != nil {
		return runtime.ReportLoader
	}
	return defaultReportLoader{}
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

func logLabelWarnings(warnings []string) {
	for _, warning := range warnings {
		log.Warn(warning)
	}
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

func resolveProvider(resolve ProviderResolver) (Provider, error) {
	if resolve == nil {
		return nil, nil
	}
	return resolve()
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
