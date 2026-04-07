package cost

import (
	"encoding/json"
	"io"
	"slices"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func outputResult(w io.Writer, workDir, outputFmt string, result *model.EstimateResult) error {
	if outputFmt == "json" {
		return outputJSONResult(w, result)
	}

	outputTextResult(workDir, result)
	return nil
}

func outputJSONResult(w io.Writer, result *model.EstimateResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func outputTextResult(workDir string, result *model.EstimateResult) {
	tree := model.BuildSegmentTree(result, workDir)
	model.CompactSegmentTree(tree)
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
	if result.TotalDiff != 0 {
		log.WithField("before", model.FormatCost(result.TotalBefore)).
			WithField("after", model.FormatCost(result.TotalAfter)).
			WithField("diff", model.FormatCostDiff(result.TotalDiff)).
			Info("total")
		return
	}

	log.WithField("monthly", model.FormatCost(result.TotalAfter)).Info("total")
}

func renderSegmentTree(node *model.SegmentNode) {
	for _, child := range node.Children {
		if !shouldShowTextSegment(child) {
			continue
		}

		if child.Module != nil && child.Module.Error != "" {
			log.WithField("error", child.Module.Error).Info(child.Name)
		} else {
			entry := log.WithField("monthly", model.FormatCost(child.AfterCost))
			if child.DiffCost != 0 {
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

func shouldShowTextSegment(node *model.SegmentNode) bool {
	if node == nil {
		return false
	}
	if node.Module != nil {
		return shouldShowTextModule(node.Module)
	}
	return node.AfterCost != 0 || node.DiffCost != 0 || hasVisibleTextDescendant(node)
}

func hasVisibleTextDescendant(node *model.SegmentNode) bool {
	return slices.ContainsFunc(node.Children, shouldShowTextSegment)
}

func shouldShowTextModule(module *model.ModuleCost) bool {
	if module == nil {
		return false
	}
	return module.BeforeCost != 0 || module.AfterCost != 0 || module.DiffCost != 0
}

func renderModuleDetails(module *model.ModuleCost) {
	if module == nil || module.Error != "" {
		return
	}

	if len(module.Submodules) > 0 {
		renderSubmodules(module.Submodules, "")
		return
	}
	renderResources(module.Resources, "")
}

func renderSubmodules(submodules []model.SubmoduleCost, parentAddr string) {
	for i := range submodules {
		submodule := &submodules[i]
		if !shouldShowSubmodule(submodule) {
			continue
		}

		showHeader := (len(submodules) > 1 || len(submodule.Children) > 0) && submodule.MonthlyCost != 0
		if showHeader && submodule.ModuleAddr != "" {
			label := model.StripModulePrefix(submodule.ModuleAddr, parentAddr)
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

func shouldShowSubmodule(submodule *model.SubmoduleCost) bool {
	if submodule == nil {
		return false
	}
	if submodule.MonthlyCost != 0 {
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

		displayAddr := model.StripModulePrefix(resource.Address, moduleAddr)
		switch resource.ErrorKind {
		case model.CostErrorNone:
			entry := log.WithField("monthly", model.FormatCost(resource.MonthlyCost))
			for _, key := range sortedDetailKeys(resource.Details) {
				entry = entry.WithField(key, resource.Details[key])
			}
			entry.Info(displayAddr)
		case model.CostErrorUsageBased:
			log.WithField("note", "usage-based").Debug(displayAddr)
		case model.CostErrorNoProvider, model.CostErrorNoHandler:
			log.WithField("note", "unsupported").Debug(displayAddr)
		case model.CostErrorLookupFailed, model.CostErrorAPIFailure, model.CostErrorNoPrice, model.CostErrorInternal:
			log.WithField("error", resource.ErrorDetail).Warn(displayAddr)
		}
	}
}

// shouldShowResource returns whether a resource should be included in text output.
// Unknown error kinds default to hidden to avoid showing unsupported resource types.
func shouldShowResource(resource *model.ResourceCost) bool {
	if resource == nil {
		return false
	}
	switch resource.ErrorKind {
	case model.CostErrorNone:
		return resource.MonthlyCost != 0 || resource.BeforeMonthlyCost != 0
	case model.CostErrorUsageBased, model.CostErrorNoProvider, model.CostErrorNoHandler,
		model.CostErrorLookupFailed, model.CostErrorAPIFailure, model.CostErrorNoPrice, model.CostErrorInternal:
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
