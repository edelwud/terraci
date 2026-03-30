package summaryengine

import "github.com/edelwud/terraci/pkg/ci"

// EnrichPlans applies per-module data from report modules to plan data.
func EnrichPlans(plans []ci.ModulePlan, modules []ci.ModuleReport) {
	if len(modules) == 0 {
		return
	}

	byPath := make(map[string]*ci.ModuleReport, len(modules))
	for i := range modules {
		byPath[modules[i].ModulePath] = &modules[i]
	}

	for i := range plans {
		if mr, ok := byPath[plans[i].ModulePath]; ok {
			if mr.HasCost {
				plans[i].CostBefore = mr.CostBefore
				plans[i].CostAfter = mr.CostAfter
				plans[i].CostDiff = mr.CostDiff
				plans[i].HasCost = true
			}
		}
	}
}
