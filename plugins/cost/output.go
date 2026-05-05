package cost

import (
	"encoding/json"
	"io"
	"slices"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/view"
)

func outputResult(w io.Writer, workDir, outputFmt string, result *model.EstimateResult) error {
	if outputFmt == "json" {
		return outputJSONResult(w, result)
	}

	// Text output uses the structured logger (pkg/log) for rich rendering.
	// The io.Writer w is intentionally unused in text mode — it exists to
	// provide a testable seam for JSON output only.
	outputTextResult(workDir, result)
	return nil
}

func outputJSONResult(w io.Writer, result *model.EstimateResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func outputTextResult(workDir string, result *model.EstimateResult) {
	tree := view.BuildSegmentTree(result, workDir)
	view.CompactSegmentTree(tree)
	renderSegmentTree(tree)
	renderSummary(result)
}

func renderSummary(result *model.EstimateResult) {
	log.Info("summary")
	log.IncreasePadding()
	defer log.DecreasePadding()

	log.WithField("count", len(result.Modules)).Info("modules")
	if len(result.Errors) > 0 {
		log.WithField("count", len(result.Errors)).Warn("errored")
	}
	if result.Unsupported > 0 {
		log.WithField("count", result.Unsupported).Warn("unsupported")
	}
	if len(result.PrefetchWarnings) > 0 {
		log.WithField("count", len(result.PrefetchWarnings)).Warn("prefetch warnings")
	}
	if result.UsageEstimated > 0 {
		log.WithField("count", result.UsageEstimated).Info("usage estimated")
	}
	if result.UsageUnknown > 0 {
		log.WithField("count", result.UsageUnknown).Warn("usage unknown")
	}
	if result.TotalDiff != 0 {
		log.WithField("before", model.FormatCost(result.TotalBefore)).
			WithField("after", model.FormatCost(result.TotalAfter)).
			WithField("diff", model.FormatCostDiff(result.TotalDiff)).
			Info("total")
		return
	}

	log.WithField("monthly", model.FormatCost(result.TotalAfter)).Info("total")
}

func renderSegmentTree(node *view.SegmentNode) {
	for _, child := range node.Children {
		if !shouldShowTextSegment(child) {
			continue
		}

		if child.Module != nil && child.Module.Error != "" {
			log.WithField("error", child.Module.Error).Info(child.Name)
		} else {
			entry := log.WithField("monthly", model.FormatCost(child.AfterCost))
			if !model.CostIsZero(child.DiffCost) {
				entry = entry.WithField("diff", model.FormatCostDiff(child.DiffCost))
			}
			entry.Info(child.Name)
		}

		if child.Module != nil {
			log.IncreasePadding()
			renderModuleDetails(child.Module)
			log.DecreasePadding()
			continue
		}

		if len(child.Children) > 0 {
			log.IncreasePadding()
			renderSegmentTree(child)
			log.DecreasePadding()
		}
	}
}

func shouldShowTextSegment(node *view.SegmentNode) bool {
	if node == nil {
		return false
	}
	if node.Module != nil {
		return shouldShowTextModule(node.Module)
	}
	return !model.CostIsZero(node.AfterCost) || !model.CostIsZero(node.DiffCost) || hasVisibleTextDescendant(node)
}

func hasVisibleTextDescendant(node *view.SegmentNode) bool {
	return slices.ContainsFunc(node.Children, shouldShowTextSegment)
}

func shouldShowTextModule(module *model.ModuleCost) bool {
	if module == nil {
		return false
	}
	return !model.CostIsZero(module.BeforeCost) || !model.CostIsZero(module.AfterCost) || !model.CostIsZero(module.DiffCost)
}

func renderModuleDetails(module *model.ModuleCost) {
	if module == nil || module.Error != "" {
		return
	}

	submodules := view.GroupByModule(module.Resources)
	if len(submodules) > 0 {
		renderSubmodules(submodules, "")
		return
	}
	renderResources(module.Resources, "")
}

func renderSubmodules(submodules []view.SubmoduleCost, parentAddr string) {
	for i := range submodules {
		submodule := &submodules[i]
		if !shouldShowSubmodule(submodule) {
			continue
		}

		showHeader := (len(submodules) > 1 || len(submodule.Children) > 0) && !model.CostIsZero(submodule.MonthlyCost)
		if showHeader && submodule.ModuleAddr != "" {
			label := view.StripModulePrefix(submodule.ModuleAddr, parentAddr)
			log.WithField("monthly", model.FormatCost(submodule.MonthlyCost)).Info(label)
			log.IncreasePadding()
		}

		renderResources(submodule.Resources, submodule.ModuleAddr)
		if len(submodule.Children) > 0 {
			renderSubmodules(submodule.Children, submodule.ModuleAddr)
		}

		if showHeader && submodule.ModuleAddr != "" {
			log.DecreasePadding()
		}
	}
}

func shouldShowSubmodule(submodule *view.SubmoduleCost) bool {
	if submodule == nil {
		return false
	}
	if !model.CostIsZero(submodule.MonthlyCost) {
		return true
	}
	for i := range submodule.Resources {
		if shouldShowResource(&submodule.Resources[i]) {
			return true
		}
	}
	for i := range submodule.Children {
		if shouldShowSubmodule(&submodule.Children[i]) {
			return true
		}
	}
	return false
}

func renderResources(resources []model.ResourceCost, moduleAddr string) {
	for i := range resources {
		resource := &resources[i]
		if !shouldShowResource(resource) {
			continue
		}

		displayAddr := view.StripModulePrefix(resource.Address, moduleAddr)
		switch resource.Status {
		case model.ResourceEstimateStatusExact:
			entry := log.WithField("monthly", model.FormatCost(resource.MonthlyCost))
			for _, key := range sortedDetailKeys(resource.Details) {
				entry = entry.WithField(key, resource.Details[key])
			}
			entry.Info(displayAddr)
		case model.ResourceEstimateStatusUsageEstimated:
			entry := log.WithField("monthly", model.FormatCost(resource.MonthlyCost)).
				WithField("note", "usage-based (estimated)")
			for _, key := range sortedDetailKeys(resource.Details) {
				entry = entry.WithField(key, resource.Details[key])
			}
			entry.Info(displayAddr)
		case model.ResourceEstimateStatusUsageUnknown:
			log.WithField("note", "usage-based (unknown)").Debug(displayAddr)
		case model.ResourceEstimateStatusUnsupported:
			log.WithField("note", "unsupported").Debug(displayAddr)
		case model.ResourceEstimateStatusFailed:
			log.WithField("error", resource.StatusDetail).Warn(displayAddr)
		default:
			log.WithField("status", resource.Status).Warn(displayAddr)
		}
	}
}

// shouldShowResource returns whether a resource should be included in text output.
// Unknown error kinds default to hidden to avoid showing unsupported resource types.
func shouldShowResource(resource *model.ResourceCost) bool {
	if resource == nil {
		return false
	}
	switch resource.Status {
	case model.ResourceEstimateStatusExact:
		return !model.CostIsZero(resource.MonthlyCost) || !model.CostIsZero(resource.BeforeMonthlyCost)
	case model.ResourceEstimateStatusUsageEstimated, model.ResourceEstimateStatusUsageUnknown,
		model.ResourceEstimateStatusUnsupported, model.ResourceEstimateStatusFailed:
		return true
	default:
		return false
	}
}

func sortedDetailKeys(details map[string]string) []string {
	keys := make([]string, 0, len(details))
	for key := range details {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
