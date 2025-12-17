package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/gitlab"
	"github.com/edelwud/terraci/pkg/log"
)

var (
	saveModuleID       string
	saveModulePath     string
	saveExitCode       int
	saveOutputFile     string
	savePlanResultsDir string
)

var savePlanResultCmd = &cobra.Command{
	Use:   "save-plan-result",
	Short: "Save terraform plan result for MR summary",
	Long: `Saves the result of a terraform plan to a JSON file that can be
collected by the summary job.

This command is typically called from the generated pipeline after
terraform plan completes. It captures the plan output and exit code
to generate the MR comment.

Exit codes from terraform plan:
  0 - Success, no changes
  1 - Error
  2 - Success, changes present`,
	RunE: runSavePlanResult,
}

func init() {
	rootCmd.AddCommand(savePlanResultCmd)

	savePlanResultCmd.Flags().StringVar(&saveModuleID, "module-id", "",
		"module identifier (e.g., platform/stage/eu-central-1/vpc)")
	savePlanResultCmd.Flags().StringVar(&saveModulePath, "module-path", "",
		"relative path to the module")
	savePlanResultCmd.Flags().IntVar(&saveExitCode, "exit-code", 0,
		"exit code from terraform plan")
	savePlanResultCmd.Flags().StringVar(&saveOutputFile, "output", "",
		"path to file containing plan output")
	savePlanResultCmd.Flags().StringVar(&savePlanResultsDir, "results-dir", gitlab.PlanResultDir,
		"directory to save plan result JSON")

	//nolint:errcheck // cobra MarkFlagRequired only fails if flag doesn't exist
	savePlanResultCmd.MarkFlagRequired("module-id")
	//nolint:errcheck // cobra MarkFlagRequired only fails if flag doesn't exist
	savePlanResultCmd.MarkFlagRequired("module-path")
}

func runSavePlanResult(_ *cobra.Command, _ []string) error {
	// Read plan output from file
	var planOutput string
	if saveOutputFile != "" {
		data, err := os.ReadFile(saveOutputFile)
		if err != nil {
			log.WithField("file", saveOutputFile).Warn("failed to read plan output file")
			// Continue without output
		} else {
			planOutput = string(data)
		}
	}

	// Create plan result writer
	writer := gitlab.NewPlanResultWriter(saveModuleID, saveModulePath, savePlanResultsDir)
	writer.SetOutput(planOutput, saveExitCode)

	if err := writer.Finish(); err != nil {
		return fmt.Errorf("failed to save plan result: %w", err)
	}

	result := writer.Result()
	log.WithField("module", saveModuleID).
		WithField("status", result.Status).
		Info("saved plan result")

	return nil
}
