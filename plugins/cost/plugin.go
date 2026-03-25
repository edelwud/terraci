// Package cost provides the AWS cost estimation plugin for TerraCi.
package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	costengine "github.com/edelwud/terraci/plugins/cost/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

// Re-export types from internal package for external consumers.
type (
	Config         = costengine.CostConfig
	EstimateResult = costengine.EstimateResult
	ModuleCost     = costengine.ModuleCost
	SubmoduleCost  = costengine.SubmoduleCost
	ResourceCost   = costengine.ResourceCost
	ErrorKind      = costengine.CostErrorKind
	Estimator      = costengine.Estimator
	SegmentNode    = costengine.SegmentNode
)

// Re-export constants from internal package.
var (
	CostErrorNone         = costengine.CostErrorNone
	CostErrorNoHandler    = costengine.CostErrorNoHandler
	CostErrorUsageBased   = costengine.CostErrorUsageBased
	CostErrorLookupFailed = costengine.CostErrorLookupFailed
	CostErrorAPIFailure   = costengine.CostErrorAPIFailure
	CostErrorNoPrice      = costengine.CostErrorNoPrice
)

// Re-export functions from internal package.
var (
	NewEstimatorFromConfig = costengine.NewEstimatorFromConfig
	DetectRegion           = costengine.DetectRegion
	BuildSegmentTree       = costengine.BuildSegmentTree
	CompactSegmentTree     = costengine.CompactSegmentTree
	StripModulePrefix      = costengine.StripModulePrefix
	FormatCost             = costengine.FormatCost
	FormatCostDiff         = costengine.FormatCostDiff
)

// Plugin is the AWS cost estimation plugin.
type Plugin struct {
	cfg        *Config
	estimator  *Estimator
	configured bool
}

func (p *Plugin) Name() string        { return "cost" }
func (p *Plugin) Description() string { return "AWS cost estimation from Terraform plans" }

// ConfigProvider

func (p *Plugin) ConfigKey() string { return "cost" }
func (p *Plugin) NewConfig() any    { return &Config{} }
func (p *Plugin) SetConfig(cfg any) error {
	cc, ok := cfg.(*Config)
	if !ok {
		return fmt.Errorf("expected *Config, got %T", cfg)
	}
	p.cfg = cc
	p.configured = true
	return nil
}

func (p *Plugin) IsConfigured() bool { return p.configured }

// Initializable — create estimator once, clean cache at startup

func (p *Plugin) Initialize(_ context.Context, appCtx *plugin.AppContext) error {
	cfg := p.effectiveConfig(appCtx)
	if !cfg.Enabled {
		return nil
	}

	log.Debug("cost: initializing estimator and pricing cache")
	p.estimator = costengine.NewEstimatorFromConfig(cfg)
	p.estimator.CleanExpiredCache()

	entries := p.estimator.CacheEntries()
	if len(entries) == 0 {
		log.WithField("dir", p.estimator.CacheDir()).Debug("pricing cache empty")
	} else {
		for _, e := range entries {
			log.WithField("service", string(e.Service)).
				WithField("region", e.Region).
				WithField("expires_in", e.ExpiresIn.Truncate(time.Minute)).
				Debug("pricing cache")
		}
	}

	return nil
}

func (p *Plugin) effectiveConfig(_ *plugin.AppContext) *Config {
	if p.cfg != nil {
		return p.cfg
	}
	return &Config{}
}

func (p *Plugin) getEstimator(cfg *Config) *Estimator {
	if p.estimator != nil {
		return p.estimator
	}
	// Fallback: create on-demand if Initialize was not called (e.g., version/schema commands)
	return costengine.NewEstimatorFromConfig(cfg)
}

// InitContributor — contributes cost estimation field to the init wizard.

const initGroupOrder = 200

func (p *Plugin) InitGroup() *plugin.InitGroupSpec {
	return &plugin.InitGroupSpec{
		Title: "Cost Estimation",
		Order: initGroupOrder,
		Fields: []plugin.InitField{
			{
				Key:         "cost.enabled",
				Title:       "Enable cost estimation?",
				Description: "Estimate AWS costs from Terraform plans",
				Type:        "bool",
				Default:     false,
			},
		},
	}
}

func (p *Plugin) BuildInitConfig(state plugin.InitState) *plugin.InitContribution {
	enabled, ok := state.Get("cost.enabled").(bool)
	if !ok {
		return nil
	}
	if !enabled {
		return nil
	}
	return &plugin.InitContribution{
		PluginKey: "cost",
		Config: map[string]any{
			"enabled":         true,
			"show_in_comment": true,
		},
	}
}

// SummaryContributor — enriches plan results with cost data during summary

func (p *Plugin) ContributeToSummary(ctx context.Context, appCtx *plugin.AppContext, execCtx *plugin.ExecutionContext) error {
	cfg := p.effectiveConfig(appCtx)
	if !cfg.Enabled {
		return nil
	}

	collection := execCtx.PlanResults
	if collection == nil || len(collection.Results) == 0 {
		return nil
	}

	// Build module paths and regions
	modulePaths := make([]string, 0, len(collection.Results))
	regions := make(map[string]string)
	for i := range collection.Results {
		r := &collection.Results[i]
		modulePaths = append(modulePaths, r.ModulePath)
		if region := r.Get("region"); region != "" {
			regions[r.ModulePath] = region
		}
	}

	est := p.getEstimator(cfg)

	// Prefetch pricing
	if err := est.ValidateAndPrefetch(ctx, modulePaths, regions); err != nil {
		return fmt.Errorf("prefetch pricing: %w", err)
	}

	// Estimate costs
	result, err := est.EstimateModules(ctx, modulePaths, regions)
	if err != nil {
		return fmt.Errorf("estimate costs: %w", err)
	}

	// Enrich plan results with cost data
	costByModule := make(map[string]int)
	for i := range result.Modules {
		costByModule[result.Modules[i].ModulePath] = i
	}
	for i := range collection.Results {
		r := &collection.Results[i]
		if idx, ok := costByModule[r.ModulePath]; ok && result.Modules[idx].Error == "" {
			mc := &result.Modules[idx]
			r.CostBefore = mc.BeforeCost
			r.CostAfter = mc.AfterCost
			r.CostDiff = mc.DiffCost
			r.HasCost = true
		}
	}

	// Store result for other plugins
	execCtx.SetData("cost:result", result)

	return nil
}

// CommandProvider

func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
	var (
		costModulePath string
		costOutputFmt  string
	)

	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Estimate AWS costs from Terraform plans",
		Long: `Estimate monthly AWS costs by analyzing plan.json files in module directories.

Examples:
  terraci cost
  terraci cost --module platform/prod/eu-central-1/rds
  terraci cost --output json`,
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx.Ensure()
			cfg := p.effectiveConfig(ctx)
			if !cfg.Enabled {
				return fmt.Errorf("cost estimation is not enabled (set plugins.cost.enabled: true)")
			}

			log.Info("running cost estimation")
			c, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			estimator := p.getEstimator(cfg)

			if costModulePath != "" {
				return p.runSingle(c, estimator, ctx, costModulePath, costOutputFmt)
			}
			return p.runAll(c, estimator, ctx, costOutputFmt)
		},
	}

	cmd.Flags().StringVarP(&costModulePath, "module", "m", "", "estimate cost for a specific module")
	cmd.Flags().StringVarP(&costOutputFmt, "output", "o", "text", "output format: text, json")

	return []*cobra.Command{cmd}
}

func (p *Plugin) runSingle(ctx context.Context, estimator *Estimator, appCtx *plugin.AppContext, modulePath, outputFmt string) error {
	fullPath := filepath.Join(appCtx.WorkDir, modulePath)
	region := costengine.DetectRegion(appCtx.Config.Structure.Segments, modulePath)

	log.WithField("module", modulePath).WithField("region", region).Info("estimating module cost")

	mc, err := estimator.EstimateModule(ctx, fullPath, region)
	if err != nil {
		return fmt.Errorf("estimate module %s: %w", modulePath, err)
	}

	return p.outputResult(appCtx, outputFmt, &EstimateResult{
		Modules:     []ModuleCost{*mc},
		TotalBefore: mc.BeforeCost,
		TotalAfter:  mc.AfterCost,
		TotalDiff:   mc.DiffCost,
		Currency:    "USD",
		GeneratedAt: time.Now().UTC(),
	})
}

func (p *Plugin) runAll(ctx context.Context, estimator *Estimator, appCtx *plugin.AppContext, outputFmt string) error {
	log.WithField("dir", appCtx.WorkDir).Info("scanning for plan.json files")

	var modulePaths []string
	regions := make(map[string]string)

	err := filepath.Walk(appCtx.WorkDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Name() == "plan.json" && !info.IsDir() {
			relDir, relErr := filepath.Rel(appCtx.WorkDir, filepath.Dir(path))
			if relErr == nil {
				fullPath := filepath.Dir(path)
				modulePaths = append(modulePaths, fullPath)
				region := costengine.DetectRegion(appCtx.Config.Structure.Segments, relDir)
				regions[fullPath] = region
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scan for plan.json: %w", err)
	}

	if len(modulePaths) == 0 {
		return fmt.Errorf("no plan.json files found")
	}

	log.WithField("count", len(modulePaths)).Info("modules with plan.json found")

	if prefetchErr := estimator.ValidateAndPrefetch(ctx, modulePaths, regions); prefetchErr != nil {
		log.WithError(prefetchErr).Warn("failed to prefetch some pricing data")
	}

	result, err := estimator.EstimateModules(ctx, modulePaths, regions)
	if err != nil {
		return fmt.Errorf("estimate costs: %w", err)
	}

	return p.outputResult(appCtx, outputFmt, result)
}

func (p *Plugin) outputResult(appCtx *plugin.AppContext, outputFmt string, result *EstimateResult) error {
	if outputFmt == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	tree := costengine.BuildSegmentTree(result, appCtx.WorkDir)
	costengine.CompactSegmentTree(tree)
	renderSegmentTree(tree, 0)

	if result.TotalDiff != 0 {
		log.WithField("before", costengine.FormatCost(result.TotalBefore)).
			WithField("after", costengine.FormatCost(result.TotalAfter)).
			WithField("diff", costengine.FormatCostDiff(result.TotalDiff)).
			Info("total")
	} else {
		log.WithField("monthly", costengine.FormatCost(result.TotalAfter)).Info("total")
	}

	return nil
}

func renderSegmentTree(node *SegmentNode, depth int) {
	for _, c := range node.Children {
		if c.AfterCost == 0 && c.DiffCost == 0 {
			continue
		}

		entry := log.WithField("monthly", costengine.FormatCost(c.AfterCost))
		if c.DiffCost != 0 {
			entry = entry.WithField("diff", costengine.FormatCostDiff(c.DiffCost))
		}
		if c.Module != nil && c.Module.Error != "" {
			entry = entry.WithField("error", c.Module.Error)
		}
		entry.Info(c.Name)

		if c.Module != nil && len(c.Module.Submodules) > 0 {
			log.IncreasePadding()
			renderSubmodules(c.Module.Submodules, "")
			log.DecreasePadding()
		}

		if len(c.Children) > 0 && c.Module == nil {
			log.IncreasePadding()
			renderSegmentTree(c, depth+1)
			log.DecreasePadding()
		}
	}
}

func renderSubmodules(submodules []SubmoduleCost, parentAddr string) {
	for i := range submodules {
		sm := &submodules[i]
		if sm.MonthlyCost == 0 && len(sm.Children) == 0 {
			continue
		}

		showHeader := len(submodules) > 1 || len(sm.Children) > 0
		if showHeader && sm.ModuleAddr != "" {
			label := costengine.StripModulePrefix(sm.ModuleAddr, parentAddr)
			log.WithField("monthly", costengine.FormatCost(sm.MonthlyCost)).Info(label)
			log.IncreasePadding()
		}

		for k := range sm.Resources {
			rc := &sm.Resources[k]
			displayAddr := costengine.StripModulePrefix(rc.Address, sm.ModuleAddr)
			renderResource(rc, displayAddr)
		}

		if len(sm.Children) > 0 {
			renderSubmodules(sm.Children, sm.ModuleAddr)
		}

		if showHeader && sm.ModuleAddr != "" {
			log.DecreasePadding()
		}
	}
}

func renderResource(rc *ResourceCost, displayAddr string) {
	switch rc.ErrorKind {
	case costengine.CostErrorNone:
		if rc.MonthlyCost > 0 {
			entry := log.WithField("monthly", costengine.FormatCost(rc.MonthlyCost))
			for dk, dv := range rc.Details {
				entry = entry.WithField(dk, dv)
			}
			entry.Info(displayAddr)
		}
	case costengine.CostErrorUsageBased:
		log.WithField("note", "usage-based").Debug(displayAddr)
	case costengine.CostErrorNoHandler:
		log.WithField("note", "unsupported").Debug(displayAddr)
	case costengine.CostErrorLookupFailed, costengine.CostErrorAPIFailure, costengine.CostErrorNoPrice:
		log.WithField("error", rc.ErrorDetail).Warn(displayAddr)
	}
}
