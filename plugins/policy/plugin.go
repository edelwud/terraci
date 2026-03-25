// Package policy provides the OPA policy check plugin for TerraCi.
package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

// Re-export types from internal package for external consumers.
type (
	Config       = policyengine.Config
	Action       = policyengine.Action
	SourceConfig = policyengine.SourceConfig
	Overwrite    = policyengine.Overwrite
	Result       = policyengine.Result
	Violation    = policyengine.Violation
	Summary      = policyengine.Summary
	Checker      = policyengine.Checker
	Engine       = policyengine.Engine
	Source       = policyengine.Source
	Puller       = policyengine.Puller
)

// Re-export constants from internal package.
var (
	ActionBlock  = policyengine.ActionBlock
	ActionWarn   = policyengine.ActionWarn
	ActionIgnore = policyengine.ActionIgnore
)

// Re-export functions from internal package.
var (
	OPAVersion = policyengine.OPAVersion
	NewChecker = policyengine.NewChecker
	NewEngine  = policyengine.NewEngine
	NewSource  = policyengine.NewSource
	NewPuller  = policyengine.NewPuller
	NewSummary = policyengine.NewSummary
)

// Plugin is the OPA policy check plugin.
type Plugin struct {
	cfg        *Config
	configured bool
}

func (p *Plugin) Name() string        { return "policy" }
func (p *Plugin) Description() string { return "OPA policy checks for Terraform plans" }

// ConfigProvider

func (p *Plugin) ConfigKey() string { return "policy" }
func (p *Plugin) NewConfig() any    { return &Config{} }
func (p *Plugin) SetConfig(cfg any) error {
	pc, ok := cfg.(*Config)
	if !ok {
		return fmt.Errorf("expected *Config, got %T", cfg)
	}
	p.cfg = pc
	p.configured = true
	return nil
}

func (p *Plugin) IsConfigured() bool { return p.configured }

// Initializable — validate OPA availability at startup

func (p *Plugin) Initialize(_ context.Context, appCtx *plugin.AppContext) error {
	cfg := p.effectiveConfig(appCtx)
	if !cfg.Enabled {
		return nil
	}

	log.WithField("opa", policyengine.OPAVersion()).Debug("policy: OPA engine available")

	// Validate policy sources are configured
	if len(cfg.Sources) == 0 {
		log.Warn("policy: enabled but no sources configured")
	}

	return nil
}

func (p *Plugin) effectiveConfig(_ *plugin.AppContext) *Config {
	if p.cfg != nil {
		return p.cfg
	}
	return &Config{}
}

// InitContributor — contributes policy check field to the init wizard.

const initGroupOrder = 201

func (p *Plugin) InitGroup() *plugin.InitGroupSpec {
	return &plugin.InitGroupSpec{
		Title: "Policy Checks",
		Order: initGroupOrder,
		Fields: []plugin.InitField{
			{
				Key:         "policy.enabled",
				Title:       "Enable policy checks?",
				Description: "Run OPA policy checks against Terraform plans",
				Type:        "bool",
				Default:     false,
			},
		},
	}
}

func (p *Plugin) BuildInitConfig(state plugin.InitState) *plugin.InitContribution {
	enabled, ok := state.Get("policy.enabled").(bool)
	if !ok {
		return nil
	}
	if !enabled {
		return nil
	}
	return &plugin.InitContribution{
		PluginKey: "policy",
		Config: map[string]any{
			"enabled": true,
		},
	}
}

// CommandProvider

func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
	var (
		policyOutput     string
		policyModulePath string
	)

	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull policies from configured sources",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx.Ensure()
			cfg := p.effectiveConfig(ctx)
			if !cfg.Enabled {
				return fmt.Errorf("policy checks are not enabled in configuration")
			}

			log.Info("pulling policies from configured sources")

			if policyOutput != "" {
				cfg.CacheDir = policyOutput
			}

			puller, err := policyengine.NewPuller(cfg, ctx.WorkDir)
			if err != nil {
				return fmt.Errorf("failed to create puller: %w", err)
			}

			c, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx.Ensure()
			cfg := p.effectiveConfig(ctx)
			if !cfg.Enabled {
				return fmt.Errorf("policy checks are not enabled in configuration")
			}

			log.Info("running policy checks")

			puller, err := policyengine.NewPuller(cfg, ctx.WorkDir)
			if err != nil {
				return fmt.Errorf("failed to create puller: %w", err)
			}

			c, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			policyDirs, err := puller.Pull(c)
			if err != nil {
				return fmt.Errorf("failed to pull policies: %w", err)
			}

			checker := policyengine.NewChecker(cfg, policyDirs, ctx.WorkDir)

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

			if err := savePolicyResults(summary); err != nil {
				log.WithError(err).Warn("failed to save policy results")
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

func savePolicyResults(summary *policyengine.Summary) error {
	dir := ".terraci"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	file, err := os.Create(dir + "/policy-results.json")
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

// PipelineContributor — adds policy-check job to CI pipeline

func (p *Plugin) PipelineSteps() []plugin.PipelineStep { return nil }

func (p *Plugin) PipelineJobs() []plugin.PipelineJob {
	if !p.IsConfigured() || p.cfg == nil || !p.cfg.Enabled {
		return nil
	}
	allowFailure := p.cfg.OnFailure == policyengine.ActionWarn
	return []plugin.PipelineJob{{
		Name:          "policy-check",
		Stage:         "post-plan",
		Commands:      []string{"terraci policy pull", "terraci policy check"},
		ArtifactPaths: []string{".terraci/policy-results.json"},
		DependsOnPlan: true,
		AllowFailure:  allowFailure,
	}}
}

// VersionProvider — contributes OPA version to `terraci version`

func (p *Plugin) VersionInfo() map[string]string {
	return map[string]string{"opa": policyengine.OPAVersion()}
}

// SummaryContributor — loads policy results and contributes to summary comment

func (p *Plugin) ContributeToSummary(_ context.Context, appCtx *plugin.AppContext, execCtx *plugin.ExecutionContext) error {
	// Try common locations for policy results
	paths := []string{
		filepath.Join(".terraci", "policy-results.json"),
		"policy-results.json",
		filepath.Join(appCtx.WorkDir, ".terraci", "policy-results.json"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var raw policyengine.Summary
		if err := json.Unmarshal(data, &raw); err != nil {
			log.WithField("path", path).WithError(err).Debug("failed to parse policy results")
			continue
		}

		// Convert to CI types and store in execution context
		policySummary := toCIPolicySummary(&raw)
		execCtx.SetData("policy:summary", policySummary)

		log.WithField("modules", policySummary.TotalModules).
			WithField("failures", policySummary.TotalFailures).
			WithField("warnings", policySummary.TotalWarnings).
			Info("loaded policy results")

		return nil
	}

	return nil
}

func toCIPolicySummary(s *policyengine.Summary) *ci.PolicySummary {
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
