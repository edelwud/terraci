package cost

import (
	"encoding/json"
	"io"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func (p *Plugin) outputResult(w io.Writer, appCtx *plugin.AppContext, outputFmt string, result *model.EstimateResult) error {
	if outputFmt == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	tree := model.BuildSegmentTree(result, appCtx.WorkDir())
	model.CompactSegmentTree(tree)
	renderSegmentTree(tree, 0)

	if result.TotalDiff != 0 {
		log.WithField("before", model.FormatCost(result.TotalBefore)).
			WithField("after", model.FormatCost(result.TotalAfter)).
			WithField("diff", model.FormatCostDiff(result.TotalDiff)).
			Info("total")
	} else {
		log.WithField("monthly", model.FormatCost(result.TotalAfter)).Info("total")
	}

	return nil
}

func renderSegmentTree(node *model.SegmentNode, depth int) {
	for _, c := range node.Children {
		if c.AfterCost == 0 && c.DiffCost == 0 {
			continue
		}

		entry := log.WithField("monthly", model.FormatCost(c.AfterCost))
		if c.DiffCost != 0 {
			entry = entry.WithField("diff", model.FormatCostDiff(c.DiffCost))
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

func renderSubmodules(submodules []model.SubmoduleCost, parentAddr string) {
	for i := range submodules {
		sm := &submodules[i]
		if sm.MonthlyCost == 0 && len(sm.Children) == 0 {
			continue
		}

		showHeader := len(submodules) > 1 || len(sm.Children) > 0
		if showHeader && sm.ModuleAddr != "" {
			label := model.StripModulePrefix(sm.ModuleAddr, parentAddr)
			log.WithField("monthly", model.FormatCost(sm.MonthlyCost)).Info(label)
			log.IncreasePadding()
		}

		for k := range sm.Resources {
			rc := &sm.Resources[k]
			displayAddr := model.StripModulePrefix(rc.Address, sm.ModuleAddr)
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

func renderResource(rc *model.ResourceCost, displayAddr string) {
	switch rc.ErrorKind {
	case model.CostErrorNone:
		if rc.MonthlyCost > 0 {
			entry := log.WithField("monthly", model.FormatCost(rc.MonthlyCost))
			for dk, dv := range rc.Details {
				entry = entry.WithField(dk, dv)
			}
			entry.Info(displayAddr)
		}
	case model.CostErrorUsageBased:
		log.WithField("note", "usage-based").Debug(displayAddr)
	case model.CostErrorNoProvider, model.CostErrorNoHandler:
		log.WithField("note", "unsupported").Debug(displayAddr)
	case model.CostErrorLookupFailed, model.CostErrorAPIFailure, model.CostErrorNoPrice:
		log.WithField("error", rc.ErrorDetail).Warn(displayAddr)
	}
}
