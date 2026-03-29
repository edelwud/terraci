package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/workflow"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

const (
	resultsFile = "update-results.json"
	reportFile  = "update-report.json"
)

// Commands returns the CLI commands provided by the update plugin.
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
	var (
		target     string
		bump       string
		write      bool
		modulePath string
		outputFmt  string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check and update Terraform dependency versions",
		Long: `Check for outdated Terraform provider and module versions.

Queries the Terraform Registry for latest versions and reports available updates.
Use --write to apply version bumps to .tf files.

Examples:
  terraci update
  terraci update --target providers --bump patch
  terraci update --write
  terraci update --module platform/prod/eu-central-1/vpc
  terraci update --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !p.IsEnabled() {
				return errors.New("update plugin is not enabled (set plugins.update.enabled: true)")
			}

			log.Info("checking dependency versions")
			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			return p.runCheck(c, ctx, cmd, write, modulePath, outputFmt)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "what to check: modules, providers, all")
	cmd.Flags().StringVarP(&bump, "bump", "b", "", "version bump level: patch, minor, major")
	cmd.Flags().BoolVarP(&write, "write", "w", false, "write updated versions back to .tf files")
	cmd.Flags().StringVarP(&modulePath, "module", "m", "", "check a specific module only")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", "text", "output format: text, json")

	return []*cobra.Command{cmd}
}

func (p *Plugin) runCheck(ctx context.Context, appCtx *plugin.AppContext, cmd *cobra.Command, write bool, modulePath, outputFmt string) error {
	baseCfg := appCtx.Config()
	workDir := appCtx.WorkDir()
	serviceDir := appCtx.ServiceDir()

	// Start from loaded config, then apply explicit CLI overrides.
	cfg := *p.Config()
	if f := cmd.Flags().Lookup("target"); f != nil && f.Changed {
		cfg.Target = f.Value.String()
	}
	if f := cmd.Flags().Lookup("bump"); f != nil && f.Changed {
		cfg.Bump = f.Value.String()
	}
	// Apply defaults for empty values.
	if cfg.Target == "" {
		cfg.Target = updateengine.TargetAll
	}
	if cfg.Bump == "" {
		cfg.Bump = updateengine.BumpMinor
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Discover modules via workflow.
	wfResult, err := workflow.Run(ctx, workflow.Options{
		WorkDir:  workDir,
		Segments: baseCfg.Structure.Segments,
		Excludes: baseCfg.Exclude,
		Includes: baseCfg.Include,
	})
	if err != nil {
		return fmt.Errorf("discover modules: %w", err)
	}

	modules := wfResult.FilteredModules
	if modulePath != "" {
		filtered := modules[:0]
		for _, m := range modules {
			if m.RelativePath == modulePath || strings.HasSuffix(m.RelativePath, modulePath) {
				filtered = append(filtered, m)
			}
		}
		modules = filtered
	}

	if len(modules) == 0 {
		return errors.New("no modules found")
	}

	log.WithField("count", len(modules)).Info("modules to check")

	tfParser := parser.NewParser(baseCfg.Structure.Segments)
	checker := updateengine.NewChecker(&cfg, tfParser, p.registry, write)

	result, err := checker.Check(ctx, modules)
	if err != nil {
		return fmt.Errorf("check versions: %w", err)
	}

	// Save artifacts.
	if serviceDir != "" {
		if saveErr := ci.SaveJSON(serviceDir, resultsFile, result); saveErr != nil {
			log.WithError(saveErr).Warn("failed to save update results")
		}
		report := buildUpdateReport(result)
		if saveErr := ci.SaveReport(serviceDir, report); saveErr != nil {
			log.WithError(saveErr).Warn("failed to save update report")
		}
	}

	return outputResult(os.Stdout, outputFmt, result)
}

func buildUpdateReport(result *updateengine.UpdateResult) *ci.Report {
	status := ci.ReportStatusPass
	if result.Summary.UpdatesAvailable > 0 {
		status = ci.ReportStatusWarn
	}

	return &ci.Report{
		Plugin:  "update",
		Title:   "Dependency Update Check",
		Status:  status,
		Summary: fmt.Sprintf("%d checked, %d updates available", result.Summary.TotalChecked, result.Summary.UpdatesAvailable),
		Body:    renderReportBody(result),
	}
}

func renderReportBody(result *updateengine.UpdateResult) string {
	var b strings.Builder

	if len(result.Providers) > 0 {
		b.WriteString("### Providers\n\n")
		b.WriteString("| Module | Provider | Current | Latest | Status |\n")
		b.WriteString("|--------|----------|---------|--------|--------|\n")
		for i := range result.Providers {
			u := &result.Providers[i]
			status := "up to date"
			if u.Skipped {
				status = u.SkipReason
			} else if u.Updated {
				status = "update available"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				u.ModulePath, u.ProviderSource, u.Constraint, u.LatestVersion, status)
		}
		b.WriteString("\n")
	}

	if len(result.Modules) > 0 {
		b.WriteString("### Modules\n\n")
		b.WriteString("| Module | Source | Current | Latest | Status |\n")
		b.WriteString("|--------|--------|---------|--------|--------|\n")
		for i := range result.Modules {
			u := &result.Modules[i]
			status := "up to date"
			if u.Skipped {
				status = u.SkipReason
			} else if u.Updated {
				status = "update available"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				u.ModulePath, u.Source, u.Constraint, u.LatestVersion, status)
		}
	}

	return b.String()
}
