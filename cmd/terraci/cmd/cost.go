package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/cost"
	"github.com/edelwud/terraci/pkg/log"
)

func newCostCmd(app *App) *cobra.Command {
	var (
		costModulePath string
		costOutputFmt  string
	)

	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Estimate AWS costs from Terraform plans",
		Long: `Estimate monthly AWS costs by analyzing plan.json files in module directories.

Scans for plan.json files (output of terraform show -json plan.tfplan),
fetches pricing from the AWS Bulk Pricing API, and calculates cost estimates.

Examples:
  terraci cost                              # Estimate all modules
  terraci cost --module platform/prod/eu-central-1/rds  # Single module
  terraci cost --output json                # JSON output`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if app.Config.Cost == nil || !app.Config.Cost.Enabled {
				log.Error("cost estimation is not enabled (set cost.enabled: true in config)")
				return fmt.Errorf("cost estimation is not enabled")
			}

			log.Info("running cost estimation")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			estimator := newCostEstimator(app)

			if costModulePath != "" {
				return runCostSingle(ctx, estimator, app, costModulePath, costOutputFmt)
			}
			return runCostAll(ctx, estimator, app, costOutputFmt)
		},
	}

	cmd.Flags().StringVarP(&costModulePath, "module", "m", "", "estimate cost for a specific module")
	cmd.Flags().StringVarP(&costOutputFmt, "output", "o", "text", "output format: text, json")

	return cmd
}

func newCostEstimator(app *App) *cost.Estimator {
	estimator := cost.NewEstimatorFromConfig(app.Config.Cost)

	// Clean up expired cache entries
	estimator.CleanExpiredCache()

	entries := estimator.CacheEntries()
	if len(entries) == 0 {
		log.WithField("dir", estimator.CacheDir()).Debug("pricing cache empty")
	} else {
		for _, e := range entries {
			log.WithField("service", string(e.Service)).
				WithField("region", e.Region).
				WithField("expires_in", e.ExpiresIn.Truncate(time.Minute)).
				Debug("pricing cache")
		}
	}

	return estimator
}

func runCostSingle(ctx context.Context, estimator *cost.Estimator, app *App, costModulePath, costOutputFmt string) error {
	modulePath := filepath.Join(app.WorkDir, costModulePath)
	region := detectRegion(app, costModulePath)

	log.WithField("module", costModulePath).WithField("region", region).Info("estimating module cost")

	mc, err := estimator.EstimateModule(ctx, modulePath, region)
	if err != nil {
		log.WithField("module", costModulePath).WithError(err).Error("estimation failed")
		return fmt.Errorf("estimate module %s: %w", costModulePath, err)
	}

	return outputCostResult(app, costOutputFmt, &cost.EstimateResult{
		Modules:     []cost.ModuleCost{*mc},
		TotalBefore: mc.BeforeCost,
		TotalAfter:  mc.AfterCost,
		TotalDiff:   mc.DiffCost,
		Currency:    "USD",
		GeneratedAt: time.Now().UTC(),
	})
}

func runCostAll(ctx context.Context, estimator *cost.Estimator, app *App, costOutputFmt string) error {
	log.WithField("dir", app.WorkDir).Info("scanning for plan.json files")

	var modulePaths []string
	regions := make(map[string]string)

	err := filepath.Walk(app.WorkDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // skip walk errors, continue scanning
		}
		if info.Name() == "plan.json" && !info.IsDir() {
			relDir, relErr := filepath.Rel(app.WorkDir, filepath.Dir(path))
			if relErr == nil {
				fullPath := filepath.Dir(path)
				modulePaths = append(modulePaths, fullPath)
				region := detectRegion(app, relDir)
				regions[fullPath] = region
				log.WithField("module", relDir).WithField("region", region).Debug("found plan.json")
			}
		}
		return nil
	})
	if err != nil {
		log.WithError(err).Error("failed to scan for plan.json")
		return fmt.Errorf("scan for plan.json: %w", err)
	}

	if len(modulePaths) == 0 {
		log.Error("no plan.json files found (run terraform plan first)")
		return fmt.Errorf("no plan.json files found")
	}

	log.WithField("count", len(modulePaths)).Info("modules with plan.json found")

	// Prefetch pricing
	log.Info("fetching AWS pricing data")
	if prefetchErr := estimator.ValidateAndPrefetch(ctx, modulePaths, regions); prefetchErr != nil {
		log.WithError(prefetchErr).Warn("failed to prefetch some pricing data")
	}

	log.Info("calculating costs")
	result, err := estimator.EstimateModules(ctx, modulePaths, regions)
	if err != nil {
		log.WithError(err).Error("cost estimation failed")
		return fmt.Errorf("estimate costs: %w", err)
	}

	return outputCostResult(app, costOutputFmt, result)
}

func outputCostResult(app *App, costOutputFmt string, result *cost.EstimateResult) error {
	if costOutputFmt == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Text output — build segment tree and render
	tree := buildSegmentTree(result, app.WorkDir)
	compactSegmentTree(tree)
	renderSegmentTree(tree, 0)

	// Total
	if result.TotalDiff != 0 {
		log.WithField("before", cost.FormatCost(result.TotalBefore)).
			WithField("after", cost.FormatCost(result.TotalAfter)).
			WithField("diff", cost.FormatCostDiff(result.TotalDiff)).
			Info("total")
	} else {
		log.WithField("monthly", cost.FormatCost(result.TotalAfter)).Info("total")
	}

	return nil
}

// segmentNode is a tree node representing one path segment (service, environment, region, or module).
type segmentNode struct {
	name      string           // segment value, e.g., "prod", "eu-central-1", "rds"
	afterCost float64          // total cost including all children
	diffCost  float64          // total diff including all children
	children  []*segmentNode   // child segments
	module    *cost.ModuleCost // non-nil only for leaf nodes (actual modules)
}

// buildSegmentTree creates a tree from module paths split by "/".
func buildSegmentTree(result *cost.EstimateResult, workDir string) *segmentNode {
	root := &segmentNode{name: ""}

	for i := range result.Modules {
		mc := &result.Modules[i]
		if mc.AfterCost == 0 && mc.BeforeCost == 0 && mc.Error == "" {
			continue
		}

		moduleID := mc.ModuleID
		if rel, err := filepath.Rel(workDir, mc.ModulePath); err == nil {
			moduleID = filepath.ToSlash(rel)
		}

		parts := strings.Split(moduleID, "/")
		node := root
		for _, part := range parts {
			child := findChild(node, part)
			if child == nil {
				child = &segmentNode{name: part}
				node.children = append(node.children, child)
			}
			child.afterCost += mc.AfterCost
			child.diffCost += mc.DiffCost
			node = child
		}
		node.module = mc
	}

	return root
}

func findChild(node *segmentNode, name string) *segmentNode {
	for _, c := range node.children {
		if c.name == name {
			return c
		}
	}
	return nil
}

// compactSegmentTree merges nodes that have exactly one child and no module into "parent/child".
func compactSegmentTree(node *segmentNode) {
	for _, c := range node.children {
		compactSegmentTree(c)
	}

	for i, c := range node.children {
		for len(c.children) == 1 && c.module == nil {
			merged := c.children[0]
			merged.name = c.name + "/" + merged.name
			node.children[i] = merged
			c = merged
		}
	}
}

// renderSegmentTree recursively renders the segment tree.
func renderSegmentTree(node *segmentNode, depth int) {
	for _, c := range node.children {
		if c.afterCost == 0 && c.diffCost == 0 {
			continue
		}

		entry := log.WithField("monthly", cost.FormatCost(c.afterCost))
		if c.diffCost != 0 {
			entry = entry.WithField("diff", cost.FormatCostDiff(c.diffCost))
		}
		if c.module != nil && c.module.Error != "" {
			entry = entry.WithField("error", c.module.Error)
		}
		entry.Info(c.name)

		// If this is a leaf module, show its terraform submodules
		if c.module != nil && len(c.module.Submodules) > 0 {
			log.IncreasePadding()
			renderSubmodules(c.module.Submodules, "")
			log.DecreasePadding()
		}

		// If this is a branch, recurse into children
		if len(c.children) > 0 && c.module == nil {
			log.IncreasePadding()
			renderSegmentTree(c, depth+1)
			log.DecreasePadding()
		}
	}
}

// detectRegion extracts region from module path using configured pattern segments.
func detectRegion(app *App, modulePath string) string {
	parts := splitPath(modulePath)
	if app.Config.Structure.Segments != nil {
		for i, seg := range app.Config.Structure.Segments {
			if seg == "region" && i < len(parts) {
				return parts[i]
			}
		}
	}
	return "us-east-1"
}

// renderSubmodules recursively renders submodule cost hierarchy.
// parentAddr is the parent's ModuleAddr, stripped from children's display names.
func renderSubmodules(submodules []cost.SubmoduleCost, parentAddr string) {
	for i := range submodules {
		sm := &submodules[i]
		if sm.MonthlyCost == 0 && len(sm.Children) == 0 {
			continue
		}

		// Show submodule header if there are multiple groups or children
		showHeader := len(submodules) > 1 || len(sm.Children) > 0
		if showHeader && sm.ModuleAddr != "" {
			label := stripModulePrefix(sm.ModuleAddr, parentAddr)
			log.WithField("monthly", cost.FormatCost(sm.MonthlyCost)).Info(label)
			log.IncreasePadding()
		}

		// Render direct resources
		for k := range sm.Resources {
			rc := &sm.Resources[k]
			displayAddr := stripModulePrefix(rc.Address, sm.ModuleAddr)
			renderResource(rc, displayAddr)
		}

		// Render children recursively
		if len(sm.Children) > 0 {
			renderSubmodules(sm.Children, sm.ModuleAddr)
		}

		if showHeader && sm.ModuleAddr != "" {
			log.DecreasePadding()
		}
	}
}

// renderResource outputs a single resource cost line.
func renderResource(rc *cost.ResourceCost, displayAddr string) {
	switch rc.ErrorKind {
	case cost.CostErrorNone:
		if rc.MonthlyCost > 0 {
			entry := log.WithField("monthly", cost.FormatCost(rc.MonthlyCost))
			for dk, dv := range rc.Details {
				entry = entry.WithField(dk, dv)
			}
			entry.Info(displayAddr)
		}
	case cost.CostErrorUsageBased:
		log.WithField("note", "usage-based").Debug(displayAddr)
	case cost.CostErrorNoHandler:
		log.WithField("note", "unsupported").Debug(displayAddr)
	case cost.CostErrorLookupFailed, cost.CostErrorAPIFailure, cost.CostErrorNoPrice:
		log.WithField("error", rc.ErrorDetail).Warn(displayAddr)
	}
}

// stripModulePrefix removes the "module.x.module.y." prefix from a resource address
// when displayed inside its module group, since it's redundant.
// e.g., "module.runner.aws_instance.web" with prefix "module.runner" → "aws_instance.web"
func stripModulePrefix(address, moduleAddr string) string {
	if moduleAddr == "" {
		return address
	}
	prefix := moduleAddr + "."
	if len(address) > len(prefix) && address[:len(prefix)] == prefix {
		return address[len(prefix):]
	}
	return address
}

func splitPath(p string) []string {
	var parts []string
	for p != "" && p != "." && p != "/" {
		dir, file := filepath.Split(p)
		if file != "" {
			parts = append([]string{file}, parts...)
		}
		p = filepath.Clean(dir)
	}
	return parts
}
