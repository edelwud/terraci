package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/ci"
	"github.com/edelwud/terraci/internal/cost"
	ghprovider "github.com/edelwud/terraci/internal/github"
	"github.com/edelwud/terraci/internal/gitlab"
	"github.com/edelwud/terraci/internal/policy"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
)

func newSummaryCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Create MR comment from plan results",
		Long: `Collects terraform plan results from artifacts and creates/updates
a summary comment on the GitLab merge request.

This command is designed to run as a final job in the pipeline after all
plan jobs have completed. It scans for plan.txt files in module directories
and posts a formatted comment to the MR.

The command automatically detects if it's running in a GitLab MR pipeline
and only creates comments when appropriate.

Environment variables:
  CI_MERGE_REQUEST_IID - MR number (auto-detected)
  CI_PROJECT_ID        - Project ID (auto-detected)
  GITLAB_TOKEN         - GitLab API token (or CI_JOB_TOKEN)

Example usage in .gitlab-ci.yml:
  terraci-summary:
    stage: summary
    script:
      - terraci summary
    needs:
      - job: plan-*
        optional: true`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSummary(app)
		},
	}

	return cmd
}

func runSummary(app *App) error {
	provider := config.ResolveProvider(app.Config)

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

	// Calculate cost estimates if enabled
	if app.Config.Cost != nil && app.Config.Cost.Enabled {
		log.Info("calculating cost estimates")
		if err := calculateCosts(app, collection); err != nil {
			log.WithError(err).Warn("failed to calculate costs, continuing without cost data")
		}
	}

	// Convert to module plans for rendering
	plans := collection.ToModulePlans()

	// Try to load policy results if they exist
	policySummary := loadPolicyResults(app)
	if policySummary != nil {
		log.WithField("modules", policySummary.TotalModules).
			WithField("failures", policySummary.TotalFailures).
			WithField("warnings", policySummary.TotalWarnings).
			Info("loaded policy results")
	}

	// Route to provider-specific comment service
	var commentSvc ci.CommentService
	switch provider {
	case config.ProviderGitHub:
		var prCfg *config.PRConfig
		if app.Config.GitHub != nil {
			prCfg = app.Config.GitHub.PR
		}
		commentSvc = ghprovider.NewPRServiceFromEnv(prCfg)
	default:
		// Check MR context first for GitLab
		mrContext := gitlab.DetectMRContext()
		if !mrContext.InMR {
			log.Info("not in MR pipeline, skipping summary")
			printSummary(collection)
			return nil
		}
		log.WithField("mr", mrContext.MRIID).Info("detected MR context")
		commentSvc = gitlab.NewMRServiceFromEnv(app.Config.GitLab.MR)
	}

	if !commentSvc.IsEnabled() {
		log.Info("PR/MR comments disabled or no token available")
		printSummary(collection)
		return nil
	}

	// Create/update comment
	log.Info("updating PR/MR comment")
	if err := commentSvc.UpsertComment(plans, policySummary); err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	log.Info("comment updated successfully")

	// Add labels if configured (GitLab only for now)
	if provider == config.ProviderGitLab && app.Config.GitLab.MR != nil && len(app.Config.GitLab.MR.Labels) > 0 {
		log.Info("adding MR labels")
		if err := addLabelsFromResults(collection); err != nil {
			log.WithField("error", err.Error()).Warn("failed to add labels")
		}
	}

	printSummary(collection)

	return nil
}

func addLabelsFromResults(collection *ci.PlanResultCollection) error {
	// Build unique labels from results
	labelSet := make(map[string]bool)

	for i := range collection.Results {
		r := &collection.Results[i]
		// Add environment-based labels
		if env := r.Get("environment"); env != "" {
			labelSet["env:"+env] = true
		}
		// Add service-based labels
		if svc := r.Get("service"); svc != "" {
			labelSet["service:"+svc] = true
		}
		// Add status-based labels
		switch r.Status {
		case ci.PlanStatusChanges:
			labelSet["terraform:changes"] = true
		case ci.PlanStatusFailed:
			labelSet["terraform:failed"] = true
		case ci.PlanStatusPending, ci.PlanStatusRunning, ci.PlanStatusSuccess, ci.PlanStatusNoChanges:
			// No label for these statuses
		}
	}

	if len(labelSet) == 0 {
		return nil
	}

	labels := make([]string, 0, len(labelSet))
	for label := range labelSet {
		labels = append(labels, label)
	}

	client := gitlab.NewClientFromEnv()
	ctx := gitlab.DetectMRContext()
	return client.AddMRLabels(ctx.ProjectID, ctx.MRIID, labels)
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

// calculateCosts calculates cost estimates for all plan results
func calculateCosts(app *App, collection *ci.PlanResultCollection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	estimator := cost.NewEstimatorFromConfig(app.Config.Cost)

	// Build module paths and regions map
	modulePaths := make([]string, 0, len(collection.Results))
	regions := make(map[string]string)

	for i := range collection.Results {
		r := &collection.Results[i]
		modulePaths = append(modulePaths, r.ModulePath)
		if region := r.Get("region"); region != "" {
			regions[r.ModulePath] = region
		}
	}

	// Validate and prefetch pricing data
	log.Info("validating pricing cache")
	if err := estimator.ValidateAndPrefetch(ctx, modulePaths, regions); err != nil {
		return fmt.Errorf("prefetch pricing: %w", err)
	}

	// Calculate costs for each module
	result, err := estimator.EstimateModules(ctx, modulePaths, regions)
	if err != nil {
		return fmt.Errorf("estimate costs: %w", err)
	}

	// Update collection results with cost data
	costByModule := make(map[string]*cost.ModuleCost)
	for i := range result.Modules {
		m := &result.Modules[i]
		costByModule[m.ModulePath] = m
	}

	for i := range collection.Results {
		r := &collection.Results[i]
		if mc, ok := costByModule[r.ModulePath]; ok && mc.Error == "" {
			r.CostBefore = mc.BeforeCost
			r.CostAfter = mc.AfterCost
			r.CostDiff = mc.DiffCost
			r.HasCost = true

			log.WithField("module", r.ModuleID).
				WithField("before", fmt.Sprintf("$%.2f", mc.BeforeCost)).
				WithField("after", fmt.Sprintf("$%.2f", mc.AfterCost)).
				WithField("diff", fmt.Sprintf("$%.2f", mc.DiffCost)).
				Debug("calculated cost")
		}
	}

	// Log total cost summary
	log.WithField("before", fmt.Sprintf("$%.2f", result.TotalBefore)).
		WithField("after", fmt.Sprintf("$%.2f", result.TotalAfter)).
		WithField("diff", fmt.Sprintf("$%.2f", result.TotalDiff)).
		Info("total cost estimate")

	return nil
}

// loadPolicyResults tries to load policy results from the artifact
func loadPolicyResults(app *App) *ci.PolicySummary {
	// Try common locations for policy results
	paths := []string{
		filepath.Join(".terraci", "policy-results.json"),
		"policy-results.json",
		filepath.Join(app.WorkDir, ".terraci", "policy-results.json"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var raw policy.Summary
		if err := json.Unmarshal(data, &raw); err != nil {
			log.WithField("path", path).WithError(err).Debug("failed to parse policy results")
			continue
		}

		return toCIPolicySummary(&raw)
	}

	return nil
}

// toCIPolicySummary converts a policy.Summary to ci.PolicySummary.
func toCIPolicySummary(s *policy.Summary) *ci.PolicySummary {
	results := make([]ci.PolicyResult, len(s.Results))
	for i, r := range s.Results {
		failures := make([]ci.PolicyViolation, len(r.Failures))
		for j, f := range r.Failures {
			failures[j] = ci.PolicyViolation{Namespace: f.Namespace, Message: f.Message}
		}
		warnings := make([]ci.PolicyViolation, len(r.Warnings))
		for j, w := range r.Warnings {
			warnings[j] = ci.PolicyViolation{Namespace: w.Namespace, Message: w.Message}
		}
		results[i] = ci.PolicyResult{
			Module:   r.Module,
			Failures: failures,
			Warnings: warnings,
		}
	}
	return &ci.PolicySummary{
		TotalModules:  s.TotalModules,
		PassedModules: s.PassedModules,
		WarnedModules: s.WarnedModules,
		FailedModules: s.FailedModules,
		TotalFailures: s.TotalFailures,
		TotalWarnings: s.TotalWarnings,
		Results:       results,
	}
}
