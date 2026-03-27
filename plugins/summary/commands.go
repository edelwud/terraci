package summary

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal"
)

// Commands returns the `terraci summary` command.
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
	return []*cobra.Command{{
		Use:   "summary",
		Short: "Create MR/PR comment from plan results",
		Long: `Collects terraform plan results from artifacts and creates/updates
a summary comment on the merge/pull request.

This command is designed to run as a final job in the pipeline after all
plan jobs have completed. It scans for plan results in module directories
and posts a formatted comment to the MR/PR.

Example:
  terraci summary`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return p.runSummary(ctx)
		},
	}}
}

func (p *Plugin) runSummary(appCtx *plugin.AppContext) error {
	// Scan plan results (provider-agnostic)
	log.Info("scanning for plan results")
	segments := []string(appCtx.Config.Structure.Segments)
	collection, err := discovery.ScanPlanResults(".", segments)
	if err != nil {
		return fmt.Errorf("failed to scan plan results: %w", err)
	}

	if len(collection.Results) == 0 {
		log.Warn("no plan results found, skipping summary")
		return nil
	}

	log.WithField("count", len(collection.Results)).Info("found plan results")

	// Convert to module plans for rendering
	plans := collection.ToModulePlans()

	// Load plugin reports from service directory
	reports := summaryengine.LoadReports(appCtx.ServiceDir)
	for _, r := range reports {
		summaryengine.EnrichPlans(plans, r.Modules)
	}

	// Check if we should skip (on_changes_only)
	if p.cfg != nil && p.cfg.OnChangesOnly && !hasReportableChanges(plans, reports) {
		log.Info("no reportable changes, skipping comment")
		printSummary(collection)
		return nil
	}

	// Resolve CI provider via plugin system (not finding a provider is not a failure)
	provider, resolveErr := plugin.ResolveProvider()
	if resolveErr != nil || provider == nil {
		log.Info("no CI provider detected, printing summary only")
		printSummary(collection)
		return nil //nolint:nilerr // intentional: no provider is gracefully handled
	}

	// Compose comment with provider metadata
	body := summaryengine.ComposeComment(plans, reports, provider.CommitSHA(), provider.PipelineID(), collection.GeneratedAt)

	commentSvc := provider.NewCommentService(appCtx)
	if !commentSvc.IsEnabled() {
		log.Info("PR/MR comments disabled or no token available")
		printSummary(collection)
		return nil
	}

	// Create/update comment
	log.Info("updating PR/MR comment")
	if upsertErr := commentSvc.UpsertComment(body); upsertErr != nil {
		return fmt.Errorf("failed to update comment: %w", upsertErr)
	}

	log.Info("comment updated successfully")
	printSummary(collection)

	return nil
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
