package planner

import (
	"maps"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/versionkit"
)

func (s *Solver) findCompatiblePlan(modulePlans []*modulePlan, providerPlans []*providerPlan) (selectedModules map[string]selectedModule, providerVersions map[string]versionkit.Version, ok bool) {
	selected := make(map[string]selectedModule, len(modulePlans))
	var search func(int) (map[string]selectedModule, map[string]versionkit.Version, bool)
	search = func(idx int) (map[string]selectedModule, map[string]versionkit.Version, bool) {
		if idx == len(modulePlans) {
			providerSelections, ok := solveProviders(providerPlans, selected, s.config.BumpPolicy())
			if !ok {
				return nil, nil, false
			}
			final := make(map[string]selectedModule, len(selected))
			maps.Copy(final, selected)
			return final, providerSelections, true
		}
		plan := modulePlans[idx]
		if len(plan.choices) == 0 {
			selected[plan.call.Name] = selectedModule{plan: plan}
			return search(idx + 1)
		}
		for _, choice := range plan.choices {
			selected[plan.call.Name] = selectedModule{plan: plan, choice: choice}
			if modulesCompatibleWithLockedProviders(providerPlans, selected) {
				if finalSelected, providerSelections, ok := search(idx + 1); ok {
					return finalSelected, providerSelections, true
				}
			}
		}
		delete(selected, plan.call.Name)
		return nil, nil, false
	}
	return search(0)
}

func modulesCompatibleWithLockedProviders(providerPlans []*providerPlan, selected map[string]selectedModule) bool {
	bySource := make(map[string]providerPlan, len(providerPlans))
	for _, plan := range providerPlans {
		bySource[plan.update.ProviderSource()] = *plan
	}
	for _, module := range selected {
		for _, dep := range module.choice.providerDeps {
			plan, ok := bySource[dep.Source]
			if !ok || !plan.hasCurrent {
				continue
			}
			constraints, err := versionkit.ParseConstraints(dep.Version)
			if err != nil {
				continue
			}
			if !versionkit.SatisfiesAll(plan.current, constraints) && len(plan.versions) == 0 {
				return false
			}
		}
	}
	return true
}

func solveProviders(providerPlans []*providerPlan, selected map[string]selectedModule, bump string) (map[string]versionkit.Version, bool) {
	result := make(map[string]versionkit.Version, len(providerPlans))
	merged := mergeProviderConstraintStrings(providerPlans, selected)
	for _, plan := range providerPlans {
		constraints, err := parseConstraintStrings(merged[plan.update.ProviderSource()])
		if err != nil {
			return nil, false
		}
		selectedVersion, ok := selectProviderVersion(plan, constraints, bump)
		if !ok {
			return nil, false
		}
		result[plan.update.ProviderSource()] = selectedVersion
	}
	return result, true
}

func mergeProviderConstraintStrings(providerPlans []*providerPlan, selected map[string]selectedModule) map[string][]string {
	result := make(map[string][]string, len(providerPlans))
	for _, plan := range providerPlans {
		result[plan.update.ProviderSource()] = append(result[plan.update.ProviderSource()], plan.baseConstraints...)
	}
	for _, module := range selected {
		for _, dep := range module.choice.providerDeps {
			if dep.Source == "" || dep.Version == "" {
				continue
			}
			result[dep.Source] = append(result[dep.Source], dep.Version)
		}
	}
	return result
}

func selectProviderVersion(plan *providerPlan, constraints []versionkit.Constraint, bump string) (versionkit.Version, bool) {
	candidates := sortVersionsDesc(plan.versions)
	for _, version := range candidates {
		if version.Prerelease != "" {
			continue
		}
		if (!plan.hasCurrent || withinBump(plan.current, version, bump)) && versionkit.SatisfiesAll(version, constraints) {
			return version, true
		}
	}
	if plan.hasCurrent && versionkit.SatisfiesAll(plan.current, constraints) {
		return plan.current, true
	}
	return versionkit.Version{}, false
}

func parseConstraintStrings(values []string) ([]versionkit.Constraint, error) {
	var result []versionkit.Constraint
	for _, value := range values {
		parsed, err := versionkit.ParseConstraints(value)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed...)
	}
	return result, nil
}
