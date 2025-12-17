package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/gitlab"
	"github.com/edelwud/terraci/pkg/log"
)

var (
	resultsDir string
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Create MR comment from plan results",
	Long: `Collects terraform plan results from artifacts and creates/updates
a summary comment on the GitLab merge request.

This command is designed to run as a final job in the pipeline after all
plan jobs have completed. It reads plan results from JSON files in the
results directory and posts a formatted comment to the MR.

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
      - terraci summary --results-dir .terraci-results
    dependencies:
      - plan-*`,
	RunE: runSummary,
}

func init() {
	rootCmd.AddCommand(summaryCmd)

	summaryCmd.Flags().StringVar(&resultsDir, "results-dir", gitlab.PlanResultDir,
		"directory containing plan result JSON files")
}

func runSummary(_ *cobra.Command, _ []string) error {
	// Check if we're in an MR context
	mrContext := gitlab.DetectMRContext()
	if !mrContext.InMR {
		log.Info("not in MR pipeline, skipping summary")
		return nil
	}

	log.WithField("mr", mrContext.MRIID).Info("detected MR context")

	// Load plan results
	log.WithField("dir", resultsDir).Info("loading plan results")
	collection, err := gitlab.LoadPlanResults(resultsDir)
	if err != nil {
		// Check if directory doesn't exist
		if os.IsNotExist(err) {
			log.Warn("no plan results found, skipping summary")
			return nil
		}
		return fmt.Errorf("failed to load plan results: %w", err)
	}

	if len(collection.Results) == 0 {
		log.Warn("no plan results found, skipping summary")
		return nil
	}

	log.WithField("count", len(collection.Results)).Info("loaded plan results")

	// Create MR service
	mrService := gitlab.NewMRService(cfg.GitLab.MR)

	if !mrService.IsEnabled() {
		log.Info("MR comments disabled or no token available")
		return nil
	}

	// Convert to module plans for rendering
	plans := collection.ToModulePlans()

	// Create/update MR comment
	log.Info("updating MR comment")
	if err := mrService.UpsertComment(plans); err != nil {
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
