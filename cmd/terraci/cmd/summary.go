package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/cost"
	"github.com/edelwud/terraci/internal/gitlab"
	"github.com/edelwud/terraci/internal/policy"
	"github.com/edelwud/terraci/pkg/log"
)

const defaultCacheTTLHours = 24

var summaryCmd = &cobra.Command{
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
	RunE: runSummary,
}

func init() {
	rootCmd.AddCommand(summaryCmd)
}

func runSummary(_ *cobra.Command, _ []string) error {
	// Check if we're in an MR context
	mrContext := gitlab.DetectMRContext()
	if !mrContext.InMR {
		log.Info("not in MR pipeline, skipping summary")
		return nil
	}

	log.WithField("mr", mrContext.MRIID).Info("detected MR context")

	// Load plan results from plan.txt files in artifacts
	log.Info("scanning for plan results")
	collection, err := gitlab.ScanPlanResults(".")
	if err != nil {
		return fmt.Errorf("failed to scan plan results: %w", err)
	}

	if len(collection.Results) == 0 {
		log.Warn("no plan results found, skipping summary")
		return nil
	}

	log.WithField("count", len(collection.Results)).Info("found plan results")

	// Create MR service
	mrService := gitlab.NewMRService(cfg.GitLab.MR)

	if !mrService.IsEnabled() {
		log.Info("MR comments disabled or no token available")
		return nil
	}

	// Calculate cost estimates if enabled
	if cfg.Cost != nil && cfg.Cost.Enabled {
		log.Info("calculating cost estimates")
		if err := calculateCosts(collection); err != nil {
			log.WithError(err).Warn("failed to calculate costs, continuing without cost data")
		}
	}

	// Convert to module plans for rendering
	plans := collection.ToModulePlans()

	// Try to load policy results if they exist
	policySummary := loadPolicyResults()
	if policySummary != nil {
		log.WithField("modules", policySummary.TotalModules).
			WithField("failures", policySummary.TotalFailures).
			WithField("warnings", policySummary.TotalWarnings).
			Info("loaded policy results")
	}

	// Create/update MR comment
	log.Info("updating MR comment")
	if err := mrService.UpsertComment(plans, policySummary); err != nil {
		return fmt.Errorf("failed to update MR comment: %w", err)
	}

	log.Info("MR comment updated successfully")

	// Add labels if configured
	if cfg.GitLab.MR != nil && len(cfg.GitLab.MR.Labels) > 0 {
		log.Info("adding MR labels")
		// Convert results to discovery modules for label expansion
		// For now, we'll use a simplified approach
		if err := addLabelsFromResults(mrService, collection); err != nil {
			log.WithField("error", err.Error()).Warn("failed to add labels")
		}
	}

	// Print summary to stdout
	printSummary(collection)

	return nil
}

func addLabelsFromResults(_ *gitlab.MRService, collection *gitlab.PlanResultCollection) error {
	// Build unique labels from results
	labelSet := make(map[string]bool)

	for i := range collection.Results {
		r := &collection.Results[i]
		// Add environment-based labels
		if r.Environment != "" {
			labelSet["env:"+r.Environment] = true
		}
		// Add service-based labels
		if r.Service != "" {
			labelSet["service:"+r.Service] = true
		}
		// Add status-based labels
		switch r.Status {
		case gitlab.PlanStatusChanges:
			labelSet["terraform:changes"] = true
		case gitlab.PlanStatusFailed:
			labelSet["terraform:failed"] = true
		case gitlab.PlanStatusPending, gitlab.PlanStatusRunning, gitlab.PlanStatusSuccess, gitlab.PlanStatusNoChanges:
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

func printSummary(collection *gitlab.PlanResultCollection) {
	var changes, noChanges, failed int
	for i := range collection.Results {
		switch collection.Results[i].Status {
		case gitlab.PlanStatusChanges:
			changes++
		case gitlab.PlanStatusNoChanges, gitlab.PlanStatusSuccess:
			noChanges++
		case gitlab.PlanStatusFailed:
			failed++
		case gitlab.PlanStatusPending, gitlab.PlanStatusRunning:
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
func calculateCosts(collection *gitlab.PlanResultCollection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Get cache settings
	cacheDir := ""
	cacheTTL := defaultCacheTTLHours * time.Hour
	if cfg.Cost.CacheDir != "" {
		cacheDir = cfg.Cost.CacheDir
	}
	if cfg.Cost.CacheTTL != "" {
		if d, err := time.ParseDuration(cfg.Cost.CacheTTL); err == nil {
			cacheTTL = d
		}
	}

	estimator := cost.NewEstimator(cacheDir, cacheTTL)

	// Build module paths and regions map
	modulePaths := make([]string, 0, len(collection.Results))
	regions := make(map[string]string)

	for i := range collection.Results {
		r := &collection.Results[i]
		modulePaths = append(modulePaths, r.ModulePath)
		if r.Region != "" {
			regions[r.ModulePath] = r.Region
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
func loadPolicyResults() *policy.Summary {
	// Try common locations for policy results
	paths := []string{
		filepath.Join(".terraci", "policy-results.json"),
		"policy-results.json",
		filepath.Join(workDir, ".terraci", "policy-results.json"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var summary policy.Summary
		if err := json.Unmarshal(data, &summary); err != nil {
			log.WithField("path", path).WithError(err).Debug("failed to parse policy results")
			continue
		}

		return &summary
	}

	return nil
}
