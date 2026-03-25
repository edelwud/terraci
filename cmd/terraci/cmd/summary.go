package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
)

func newSummaryCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Create MR/PR comment from plan results",
		Long: `Collects terraform plan results from artifacts and creates/updates
a summary comment on the merge/pull request.

This command is designed to run as a final job in the pipeline after all
plan jobs have completed. It scans for plan results in module directories
and posts a formatted comment to the MR/PR.

Example:
  terraci summary`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSummary(cmd, app)
		},
	}

	return cmd
}

func runSummary(cmd *cobra.Command, app *App) error {
	// Scan plan results (provider-agnostic)
	log.Info("scanning for plan results")
	segments := []string(app.Config.Structure.Segments)
	collection, err := ci.ScanPlanResults(".", segments)
	if err != nil {
		return fmt.Errorf("failed to scan plan results: %w", err)
	}

	if len(collection.Results) == 0 {
		log.Warn("no plan results found, skipping summary")
		return nil
	}

	log.WithField("count", len(collection.Results)).Info("found plan results")

	// Create execution context for plugin contributions
	execCtx := plugin.NewExecutionContext(collection)

	// Let all SummaryContributor plugins enrich the data
	for _, c := range plugin.ByCapability[plugin.SummaryContributor]() {
		if contributeErr := c.ContributeToSummary(cmd.Context(), app.PluginContext(), execCtx); contributeErr != nil {
			log.WithError(contributeErr).WithField("plugin", c.Name()).Warn("summary contribution failed")
		}
	}

	// Convert to module plans for rendering
	plans := collection.ToModulePlans()

	// Resolve policy summary from execution context (if contributed by policy plugin)
	var policySummary *ci.PolicySummary
	if raw, ok := execCtx.GetData("policy:summary"); ok {
		if ps, ok := raw.(*ci.PolicySummary); ok {
			policySummary = ps
		}
	}

	// Resolve CI provider via plugin system (not finding a provider is not a failure)
	provider, resolveErr := plugin.ResolveProvider()
	if resolveErr != nil || provider == nil {
		log.Info("no CI provider detected, printing summary only")
		printSummary(collection)
		return nil //nolint:nilerr // intentional: no provider is gracefully handled
	}

	commentSvc := provider.NewCommentService(app.PluginContext())
	if !commentSvc.IsEnabled() {
		log.Info("PR/MR comments disabled or no token available")
		printSummary(collection)
		return nil
	}

	// Create/update comment
	log.Info("updating PR/MR comment")
	if upsertErr := commentSvc.UpsertComment(plans, policySummary); upsertErr != nil {
		return fmt.Errorf("failed to update comment: %w", upsertErr)
	}

	log.Info("comment updated successfully")
	printSummary(collection)

	return nil
}

func printSummary(collection *ci.PlanResultCollection) {
	var changes, noChanges, failed int
	for i := range collection.Results {
		switch collection.Results[i].Status {
		case ci.PlanStatusChanges:
			changes++
		case ci.PlanStatusNoChanges, ci.PlanStatusSuccess:
			noChanges++
		case ci.PlanStatusFailed:
			failed++
		case ci.PlanStatusPending, ci.PlanStatusRunning:
			// Not counted
		}
	}

	log.Info("summary")
	log.IncreasePadding()
	log.WithField("total", len(collection.Results)).Info("modules")
	if changes > 0 {
		log.WithField("count", changes).Info("with changes")
	}
	if noChanges > 0 {
		log.WithField("count", noChanges).Info("no changes")
	}
	if failed > 0 {
		log.WithField("count", failed).Warn("failed")
	}
	log.DecreasePadding()
}
