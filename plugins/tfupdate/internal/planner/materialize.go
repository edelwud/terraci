package planner

import (
	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/versionkit"
)

func (s *Solver) materializeTargetModuleResolutions(scanCtx *moduleScanContext, modulePlans []*modulePlan, selected map[string]selectedModule, compatible bool) []domain.ModuleResolution {
	if !s.config.ShouldCheckModules() {
		return nil
	}
	return s.materializeModuleResolutions(scanCtx, modulePlans, selected, compatible)
}

func (s *Solver) materializeTargetProviderResolutions(scanCtx *moduleScanContext, providerPlans []*providerPlan, selections map[string]versionkit.Version, compatible bool) []domain.ProviderResolution {
	if !s.config.ShouldCheckProviders() {
		return nil
	}
	return s.materializeProviderResolutions(scanCtx, providerPlans, selections, compatible)
}

func (s *Solver) materializeModuleResolutions(scanCtx *moduleScanContext, modulePlans []*modulePlan, selected map[string]selectedModule, compatible bool) []domain.ModuleResolution {
	resolutions := make([]domain.ModuleResolution, 0, len(modulePlans))
	for _, plan := range modulePlans {
		resolution := domain.ModuleResolution{
			Dependency: plan.update.Dependency,
			Registry: domain.RegistrySelection{
				Hostname: plan.address.Hostname,
				Source:   plan.address.Source(),
			},
			File: scanCtx.findModuleFile(plan.call.Name),
		}
		if !plan.current.IsZero() {
			resolution.Current = plan.current.String()
		}
		if !plan.latest.IsZero() {
			resolution.Latest = plan.latest.String()
		}
		if plan.update.Status == domain.StatusError || plan.update.Status == domain.StatusSkipped {
			resolution.Status = plan.update.Status
			resolution.Issue = plan.update.Issue
			resolutions = append(resolutions, resolution)
			continue
		}
		chosen, ok := selected[plan.call.Name]
		if ok {
			resolution.ProviderDeps = chosen.choice.providerDeps
			if !chosen.choice.version.IsZero() {
				resolution.Selected = chosen.choice.version.String()
			}
			if !chosen.choice.version.IsZero() && chosen.choice.version.Compare(plan.current) > 0 {
				resolution.Status = domain.StatusUpdateAvailable
			} else {
				resolution.Status = domain.StatusUpToDate
			}
		}
		if !compatible && plan.updateChoices > 0 {
			resolution.Status = domain.StatusSkipped
			resolution.Issue = "no compatible provider resolution for module update policy"
		}
		resolutions = append(resolutions, resolution)
	}
	return resolutions
}

func (s *Solver) materializeProviderResolutions(scanCtx *moduleScanContext, providerPlans []*providerPlan, selections map[string]versionkit.Version, compatible bool) []domain.ProviderResolution {
	resolutions := make([]domain.ProviderResolution, 0, len(providerPlans))
	for _, plan := range providerPlans {
		file := ""
		if plan.required != nil {
			file = scanCtx.findProviderFile(plan.required.Name)
		}
		resolution := domain.ProviderResolution{
			Dependency: plan.update.Dependency,
			Registry: domain.RegistrySelection{
				Hostname: plan.address.Hostname,
				Source:   plan.address.Source(),
			},
			File: file,
		}
		if plan.hasCurrent {
			resolution.Current = plan.current.String()
		}
		if !plan.latest.IsZero() {
			resolution.Latest = plan.latest.String()
		}
		selected, ok := selections[plan.update.ProviderSource()]
		resolution.Constraints.Raw = append(resolution.Constraints.Raw, plan.baseConstraints...)
		if plan.locked != nil {
			resolution.Locked = true
			resolution.LockedSource = plan.locked.Source
			resolution.LockedConstraint = plan.locked.Constraints
		}
		switch {
		case plan.update.Status == domain.StatusSkipped || plan.update.Status == domain.StatusError:
			resolution.Status = plan.update.Status
			resolution.Issue = plan.update.Issue
		case !compatible || !ok:
			resolution.Status = domain.StatusSkipped
			resolution.Issue = "no provider version satisfies merged direct and transitive constraints"
		case plan.hasCurrent && selected.Compare(plan.current) > 0:
			resolution.Selected = selected.String()
			resolution.Status = domain.StatusUpdateAvailable
		case !plan.hasCurrent:
			resolution.Status = domain.StatusSkipped
			resolution.Issue = "cannot determine current version"
		default:
			resolution.Status = domain.StatusUpToDate
		}
		resolutions = append(resolutions, resolution)
	}
	return resolutions
}
