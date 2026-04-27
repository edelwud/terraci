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

const summaryPluginName = "summary"

type summaryProvider interface {
	CommitSHA() string
	PipelineID() string
	NewCommentService(ctx *plugin.AppContext) (ci.CommentService, bool)
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
	reports, err := ci.LoadReports(serviceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin reports: %w", err)
	}
	filteredReports := reports[:0]
	for _, r := range reports {
		if r == nil || r.Plugin == summaryPluginName {
			continue
		}
		filteredReports = append(filteredReports, r)
	}
	summaryengine.EnrichPlansFromReports(plans, filteredReports)

	return &summaryInputs{
		collection: collection,
		plans:      plans,
		reports:    filteredReports,
	}, nil
}

func resolveSummaryProvider(appCtx *plugin.AppContext) func() (summaryProvider, error) {
	return func() (summaryProvider, error) {
		return plugin.ResolveCIProvider(appCtx)
	}
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

	provider, resolveErr := resolveProvider()
	commitSHA, pipelineID := "", ""
	if resolveErr == nil && provider != nil {
		commitSHA = provider.CommitSHA()
		pipelineID = provider.PipelineID()
	}
	if err := saveSummaryReport(appCtx, inputs, cfg, commitSHA, pipelineID); err != nil {
		return err
	}

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
	return runSummaryUseCase(ctx, appCtx, p.Config(), resolveSummaryProvider(appCtx))
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

func saveSummaryReport(appCtx *plugin.AppContext, inputs *summaryInputs, cfg *summaryengine.Config, commitSHA, pipelineID string) error {
	if inputs != nil && inputs.collection != nil {
		inputs.collection.CommitSHA = commitSHA
		inputs.collection.PipelineID = pipelineID
	}
	report := buildSummaryReport(inputs, cfg)
	if err := ci.SaveReport(appCtx.ServiceDir(), report); err != nil {
		return fmt.Errorf("save summary report: %w", err)
	}
	return nil
}

func buildSummaryReport(inputs *summaryInputs, cfg *summaryengine.Config) *ci.Report {
	return &ci.Report{
		Plugin:  summaryPluginName,
		Title:   "Terraform Plan Summary",
		Status:  summaryReportStatus(inputs.plans, inputs.reports),
		Summary: summaryReportSummary(inputs.collection),
		Provenance: &ci.ReportProvenance{
			Producer:               summaryPluginName,
			CommitSHA:              inputs.collection.CommitSHA,
			PipelineID:             inputs.collection.PipelineID,
			PlanResultsFingerprint: inputs.collection.Fingerprint(),
		},
		Sections: summaryengine.BuildSummarySectionsWithOptions(inputs.plans, inputs.reports, summaryIncludeDetails(cfg)),
	}
}

func summaryIncludeDetails(cfg *summaryengine.Config) bool {
	if cfg == nil || cfg.IncludeDetails == nil {
		return true
	}
	return *cfg.IncludeDetails
}

func summaryReportStatus(plans []ci.ModulePlan, reports []*ci.Report) ci.ReportStatus {
	for i := range plans {
		if plans[i].Status == ci.PlanStatusFailed {
			return ci.ReportStatusFail
		}
	}
	for _, report := range reports {
		if report.Status == ci.ReportStatusFail {
			return ci.ReportStatusFail
		}
	}
	for i := range plans {
		if plans[i].Status == ci.PlanStatusChanges {
			return ci.ReportStatusWarn
		}
	}
	for _, report := range reports {
		if report.Status == ci.ReportStatusWarn {
			return ci.ReportStatusWarn
		}
	}
	return ci.ReportStatusPass
}

func summaryReportSummary(collection *ci.PlanResultCollection) string {
	var changes, noChanges, failed, pending, running int
	for i := range collection.Results {
		switch collection.Results[i].Status {
		case ci.PlanStatusChanges:
			changes++
		case ci.PlanStatusNoChanges, ci.PlanStatusSuccess:
			noChanges++
		case ci.PlanStatusFailed:
			failed++
		case ci.PlanStatusPending:
			pending++
		case ci.PlanStatusRunning:
			running++
		}
	}

	summary := fmt.Sprintf(
		"%d modules: %d with changes, %d no changes, %d failed",
		len(collection.Results),
		changes,
		noChanges,
		failed,
	)
	if pending > 0 {
		summary += fmt.Sprintf(", %d pending", pending)
	}
	if running > 0 {
		summary += fmt.Sprintf(", %d running", running)
	}
	return summary
}
