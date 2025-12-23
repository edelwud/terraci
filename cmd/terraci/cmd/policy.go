package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/policy"
	"github.com/edelwud/terraci/pkg/log"
)

var (
	policyOutput     string
	policyModulePath string
)

// policyPullCmd pulls policies from configured sources
var policyPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull policies from configured sources",
	Long: `Pull policies from configured sources (local path, git, or OCI).

Policies are downloaded to the cache directory (default: .terraci/policies).
This command should be run before 'terraci policy check'.

Example:
  terraci policy pull
  terraci policy pull --output ./my-policies`,
	RunE: runPolicyPull,
}

// policyCheckCmd checks Terraform plans against policies
var policyCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check Terraform plans against policies",
	Long: `Check Terraform plans against OPA/Rego policies.

This command scans for plan.json files in module directories and
evaluates them against the configured policies.

Policies must be pulled first using 'terraci policy pull'.

Example:
  terraci policy check
  terraci policy check --module platform/prod/eu-central-1/vpc
  terraci policy check --output json`,
	RunE: runPolicyCheck,
}

func init() {
	// Create policy parent command
	policyCmd := &cobra.Command{
		Use:   "policy",
		Short: "Policy management commands",
		Long:  "Commands for managing and running OPA policy checks against Terraform plans.",
	}

	// Add subcommands
	policyCmd.AddCommand(policyPullCmd)
	policyCmd.AddCommand(policyCheckCmd)

	// Add to root
	rootCmd.AddCommand(policyCmd)

	// Flags for pull
	policyPullCmd.Flags().StringVarP(&policyOutput, "output", "o", "", "output directory for policies (overrides config)")

	// Flags for check
	policyCheckCmd.Flags().StringVarP(&policyModulePath, "module", "m", "", "check specific module only")
	policyCheckCmd.Flags().StringVarP(&policyOutput, "output", "o", "", "output format: text, json (default: text)")
}

func runPolicyPull(_ *cobra.Command, _ []string) error {
	if cfg.Policy == nil || !cfg.Policy.Enabled {
		return fmt.Errorf("policy checks are not enabled in configuration")
	}

	log.Info("pulling policies from configured sources")

	// Override cache dir if output specified
	if policyOutput != "" {
		cfg.Policy.CacheDir = policyOutput
	}

	puller, err := policy.NewPuller(cfg.Policy, workDir)
	if err != nil {
		return fmt.Errorf("failed to create puller: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	dirs, err := puller.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull policies: %w", err)
	}

	log.WithField("count", len(dirs)).Info("policy sources pulled")
	for _, dir := range dirs {
		log.WithField("path", dir).Debug("policy directory")
	}

	log.WithField("cache", puller.CacheDir()).Info("policies cached")
	return nil
}

func runPolicyCheck(_ *cobra.Command, _ []string) error {
	if cfg.Policy == nil || !cfg.Policy.Enabled {
		return fmt.Errorf("policy checks are not enabled in configuration")
	}

	log.Info("running policy checks")

	// Get policy directories
	puller, err := policy.NewPuller(cfg.Policy, workDir)
	if err != nil {
		return fmt.Errorf("failed to create puller: %w", err)
	}

	// Pull policies (in case not pulled yet)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	policyDirs, err := puller.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull policies: %w", err)
	}

	// Create checker
	checker := policy.NewChecker(cfg.Policy, policyDirs, workDir)

	var summary *policy.Summary

	if policyModulePath != "" {
		// Check single module
		result, checkErr := checker.CheckModule(ctx, policyModulePath)
		if checkErr != nil {
			return fmt.Errorf("policy check failed: %w", checkErr)
		}
		summary = policy.NewSummary([]policy.Result{*result})
	} else {
		// Check all modules
		var checkErr error
		summary, checkErr = checker.CheckAll(ctx)
		if checkErr != nil {
			return fmt.Errorf("policy check failed: %w", checkErr)
		}
	}

	// Save results to artifact file for summary job
	if err := savePolicyResults(summary); err != nil {
		log.WithError(err).Warn("failed to save policy results")
	}

	// Output results
	if policyOutput == "json" {
		return outputJSON(summary)
	}

	return outputText(summary, checker.ShouldBlock(summary))
}

// savePolicyResults saves the policy results to a JSON file for the summary job
func savePolicyResults(summary *policy.Summary) error {
	// Create .terraci directory if it doesn't exist
	dir := ".terraci"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write results to file
	path := dir + "/policy-results.json"
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		return fmt.Errorf("failed to encode results: %w", err)
	}

	return nil
}

func outputJSON(summary *policy.Summary) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

func outputText(summary *policy.Summary, shouldBlock bool) error {
	// Print summary
	log.WithField("total", summary.TotalModules).
		WithField("passed", summary.PassedModules).
		WithField("warned", summary.WarnedModules).
		WithField("failed", summary.FailedModules).
		Info("policy check summary")

	// Print details for failed/warned modules
	for _, result := range summary.Results {
		if result.Status() == "pass" {
			continue
		}

		log.WithField("module", result.Module).
			WithField("status", result.Status()).
			Info("module result")

		log.IncreasePadding()

		for _, f := range result.Failures {
			log.WithField("namespace", f.Namespace).
				WithField("message", f.Message).
				Error("failure")
		}

		for _, w := range result.Warnings {
			log.WithField("namespace", w.Namespace).
				WithField("message", w.Message).
				Warn("warning")
		}

		log.DecreasePadding()
	}

	// Final status
	if shouldBlock {
		log.Error("policy check FAILED")
		return fmt.Errorf("policy check failed with %d failures", summary.TotalFailures)
	}

	if summary.HasWarnings() {
		log.Warn("policy check passed with warnings")
	} else {
		log.Info("policy check PASSED")
	}

	return nil
}
