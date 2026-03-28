package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

// Commands returns the CLI commands provided by the policy plugin.
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
	var (
		policyOutput     string
		policyModulePath string
	)

	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull policies from configured sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !p.IsConfigured() {
				return fmt.Errorf("policy checks are not enabled in configuration")
			}

			log.Info("pulling policies from configured sources")

			cfg := p.cfg
			if policyOutput != "" {
				cfg.CacheDir = policyOutput
			}

			puller, err := policyengine.NewPuller(cfg, ctx.WorkDir, ctx.ServiceDir)
			if err != nil {
				return fmt.Errorf("failed to create puller: %w", err)
			}

			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			dirs, err := puller.Pull(c)
			if err != nil {
				return fmt.Errorf("failed to pull policies: %w", err)
			}

			log.WithField("count", len(dirs)).Info("policy sources pulled")
			log.WithField("cache", puller.CacheDir()).Info("policies cached")
			return nil
		},
	}

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check Terraform plans against policies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !p.IsConfigured() {
				return fmt.Errorf("policy checks are not enabled in configuration")
			}

			log.Info("running policy checks")

			puller, err := policyengine.NewPuller(p.cfg, ctx.WorkDir, ctx.ServiceDir)
			if err != nil {
				return fmt.Errorf("failed to create puller: %w", err)
			}

			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			policyDirs, err := puller.Pull(c)
			if err != nil {
				return fmt.Errorf("failed to pull policies: %w", err)
			}

			checker := policyengine.NewChecker(p.cfg, policyDirs, ctx.WorkDir)

			var summary *policyengine.Summary

			if policyModulePath != "" {
				result, checkErr := checker.CheckModule(c, policyModulePath)
				if checkErr != nil {
					return fmt.Errorf("policy check failed: %w", checkErr)
				}
				summary = policyengine.NewSummary([]policyengine.Result{*result})
			} else {
				var checkErr error
				summary, checkErr = checker.CheckAll(c)
				if checkErr != nil {
					return fmt.Errorf("policy check failed: %w", checkErr)
				}
			}

			if ctx.ServiceDir != "" {
				if saveErr := ci.SaveJSON(ctx.ServiceDir, resultsFile, summary); saveErr != nil {
					log.WithError(saveErr).Warn("failed to save policy results")
				}
				report := buildPolicyReport(summary)
				if saveErr := ci.SaveReport(ctx.ServiceDir, report); saveErr != nil {
					log.WithError(saveErr).Warn("failed to save policy report")
				}
			}

			if policyOutput == "json" {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(summary)
			}

			return outputText(summary, checker.ShouldBlock(summary))
		},
	}

	pullCmd.Flags().StringVarP(&policyOutput, "output", "o", "", "output directory for policies")
	checkCmd.Flags().StringVarP(&policyModulePath, "module", "m", "", "check specific module only")
	checkCmd.Flags().StringVarP(&policyOutput, "output", "o", "", "output format: text, json")

	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Policy management commands",
		Long:  "Commands for managing and running OPA policy checks against Terraform plans.",
	}
	cmd.AddCommand(pullCmd, checkCmd)

	return []*cobra.Command{cmd}
}

func outputText(summary *policyengine.Summary, shouldBlock bool) error {
	log.WithField("total", summary.TotalModules).
		WithField("passed", summary.PassedModules).
		WithField("warned", summary.WarnedModules).
		WithField("failed", summary.FailedModules).
		Info("policy check summary")

	for _, result := range summary.Results {
		if result.Status() == "pass" {
			continue
		}
		log.WithField("module", result.Module).WithField("status", result.Status()).Info("module result")
		log.IncreasePadding()
		for _, f := range result.Failures {
			log.WithField("namespace", f.Namespace).WithField("message", f.Message).Error("failure")
		}
		for _, w := range result.Warnings {
			log.WithField("namespace", w.Namespace).WithField("message", w.Message).Warn("warning")
		}
		log.DecreasePadding()
	}

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

func buildPolicyReport(summary *policyengine.Summary) *ci.Report {
	status := ci.ReportStatusPass
	if summary.FailedModules > 0 {
		status = ci.ReportStatusFail
	} else if summary.WarnedModules > 0 {
		status = ci.ReportStatusWarn
	}

	return &ci.Report{
		Plugin:  "policy",
		Title:   "Policy Check",
		Status:  status,
		Summary: fmt.Sprintf("%d modules: %d passed, %d warned, %d failed", summary.TotalModules, summary.PassedModules, summary.WarnedModules, summary.FailedModules),
		Body:    renderPolicyReportBody(summary),
	}
}

func renderPolicyReportBody(summary *policyengine.Summary) string {
	var b strings.Builder
	for _, r := range summary.Results {
		if r.Status() == "pass" {
			continue
		}
		fmt.Fprintf(&b, "**%s** (%s)\n", r.Module, r.Status())
		for _, f := range r.Failures {
			fmt.Fprintf(&b, "- :x: %s", f.Message)
			if f.Namespace != "" {
				fmt.Fprintf(&b, " (%s)", f.Namespace)
			}
			b.WriteString("\n")
		}
		for _, w := range r.Warnings {
			fmt.Fprintf(&b, "- :warning: %s", w.Message)
			if w.Namespace != "" {
				fmt.Fprintf(&b, " (%s)", w.Namespace)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}
