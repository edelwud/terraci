package summary

import (
	"context"
	"fmt"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/planresults"
	"github.com/edelwud/terraci/pkg/plugin"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal"
)

type summaryProvider interface {
	CommitSHA() string
	PipelineID() string
	NewCommentService(ctx *plugin.AppContext) (ci.CommentService, bool)
}

type summaryInputs struct {
	collection *ci.PlanResultCollection
	plans      []ci.PlanResult
	reports    []*ci.Report
}

func loadSummaryInputs(appCtx *plugin.AppContext) (*summaryInputs, error) {
	cfg := appCtx.Config()
	workDir := appCtx.WorkDir()
	serviceDir := appCtx.ServiceDir()

	log.Info("scanning for plan results")
	segments := []string(cfg.Structure.Segments)
	collection, err := planresults.Scan(workDir, segments)
	if err != nil {
		return nil, fmt.Errorf("failed to scan plan results: %w", err)
	}

	if len(collection.Results) == 0 {
		return &summaryInputs{collection: collection}, nil
	}

	log.WithField("count", len(collection.Results)).Info("found plan results")

	plans := collection.Results
	reports, err := ci.LoadReports(serviceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin reports: %w", err)
	}
	filteredReports := reports[:0]
	for _, r := range reports {
		if r == nil || r.Producer == pluginName {
			continue
		}
		filteredReports = append(filteredReports, r)
	}
	return &summaryInputs{
		collection: collection,
		plans:      plans,
		reports:    filteredReports,
	}, nil
}

func resolveSummaryProvider(appCtx *plugin.AppContext) func() (summaryProvider, error) {
	return func() (summaryProvider, error) {
		return appCtx.Resolver().ResolveCIProvider()
	}
}

func runSummaryUseCase(ctx context.Context, appCtx *plugin.AppContext, runtime *summaryRuntime) error {
	if runtime == nil {
		runtime = newRuntime(appCtx, nil)
	}
	cfg := runtime.config
	if cfg == nil {
		cfg = &summaryengine.Config{}
	}
	resolveProvider := runtime.resolveProvider
	if resolveProvider == nil {
		resolveProvider = resolveSummaryProvider(appCtx)
	}

	inputs, err := loadSummaryInputs(appCtx)
	if err != nil {
		return err
	}
	if len(inputs.collection.Results) == 0 {
		log.Warn("no plan results found, skipping summary")
		return nil
	}

	provider, resolveErr := resolveProvider()
	if cfg.OnChangesOnly && !hasReportableChanges(inputs.plans, inputs.reports) {
		log.Info("no reportable changes, skipping comment")
		printSummary(inputs.collection)
		return nil
	}

	if resolveErr != nil || provider == nil {
		log.Info("no CI provider detected, printing summary only")
		printSummary(inputs.collection)
		return nil //nolint:nilerr // intentional: no provider is gracefully handled
	}

	body := summaryengine.ComposeCommentWithOptions(
		inputs.plans,
		inputs.reports,
		provider.CommitSHA(),
		provider.PipelineID(),
		inputs.collection.GeneratedAt,
		summaryIncludeDetails(cfg),
	)

	commentSvc, ok := provider.NewCommentService(appCtx)
	if !ok || !commentSvc.IsEnabled() {
		log.Info("PR/MR comments disabled or no token available")
		printSummary(inputs.collection)
		return nil
	}

	log.Info("updating PR/MR comment")
	if err := commentSvc.UpsertComment(ctx, body); err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	log.Info("comment updated successfully")
	printSummary(inputs.collection)
	return nil
}

func (p *Plugin) runSummary(ctx context.Context, appCtx *plugin.AppContext) error {
	runtime, err := p.runtime(ctx, appCtx)
	if err != nil {
		return err
	}
	return runSummaryUseCase(ctx, appCtx, runtime)
}

func hasReportableChanges(plans []ci.PlanResult, reports []*ci.Report) bool {
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

func summaryIncludeDetails(cfg *summaryengine.Config) bool {
	if cfg == nil || cfg.IncludeDetails == nil {
		return true
	}
	return *cfg.IncludeDetails
}
