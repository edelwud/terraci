package summary

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal"
)

type summaryProvider interface {
	CommitSHA() string
	PipelineID() string
	NewCommentService(ctx *plugin.AppContext) ci.CommentService
}

type summaryInputs struct {
	collection *ci.PlanResultCollection
	plans      []ci.ModulePlan
	reports    []*ci.Report
}

func loadSummaryInputs(appCtx *plugin.AppContext) (*summaryInputs, error) {
	cfg := appCtx.Config()
	workDir := appCtx.WorkDir()
	serviceDir := appCtx.ServiceDir()

	log.Info("scanning for plan results")
	segments := []string(cfg.Structure.Segments)
	collection, err := discovery.ScanPlanResults(workDir, segments)
	if err != nil {
		return nil, fmt.Errorf("failed to scan plan results: %w", err)
	}

	if len(collection.Results) == 0 {
		return &summaryInputs{collection: collection}, nil
	}

	log.WithField("count", len(collection.Results)).Info("found plan results")

	plans := collection.ToModulePlans()
	reports := summaryengine.LoadReports(serviceDir)
	for _, r := range reports {
		summaryengine.EnrichPlans(plans, r.Modules)
	}

	return &summaryInputs{
		collection: collection,
		plans:      plans,
		reports:    reports,
	}, nil
}

func resolveSummaryProvider() (summaryProvider, error) {
	return plugin.ResolveProvider()
}

func runSummaryUseCase(ctx context.Context, appCtx *plugin.AppContext, cfg *summaryengine.Config, resolveProvider func() (summaryProvider, error)) error {
	if cfg == nil {
		cfg = &summaryengine.Config{}
	}

	inputs, err := loadSummaryInputs(appCtx)
	if err != nil {
		return err
	}
	if len(inputs.collection.Results) == 0 {
		log.Warn("no plan results found, skipping summary")
		return nil
	}

	if cfg.OnChangesOnly && !hasReportableChanges(inputs.plans, inputs.reports) {
		log.Info("no reportable changes, skipping comment")
		printSummary(inputs.collection)
		return nil
	}

	provider, resolveErr := resolveProvider()
	if resolveErr != nil || provider == nil {
		log.Info("no CI provider detected, printing summary only")
		printSummary(inputs.collection)
		return nil //nolint:nilerr // intentional: no provider is gracefully handled
	}

	body := summaryengine.ComposeComment(
		inputs.plans,
		inputs.reports,
		provider.CommitSHA(),
		provider.PipelineID(),
		inputs.collection.GeneratedAt,
	)

	commentSvc := provider.NewCommentService(appCtx)
	if !commentSvc.IsEnabled() {
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
	return runSummaryUseCase(ctx, appCtx, p.Config(), resolveSummaryProvider)
}

func hasReportableChanges(plans []ci.ModulePlan, reports []*ci.Report) bool {
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
