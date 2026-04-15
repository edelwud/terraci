package planner

import (
	"fmt"
	"sort"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/versionkit"
)

func (s *Solver) buildLockSyncPlan(
	scanCtx *moduleScanContext,
	providerPlans []*providerPlan,
	selected map[string]selectedModule,
	selections map[string]versionkit.Version,
	compatible bool,
) (domain.LockSyncPlan, error) {
	plan := domain.LockSyncPlan{ModulePath: scanCtx.module.RelativePath}
	if !compatible {
		return plan, nil
	}

	mergedConstraints := mergeProviderConstraintStrings(providerPlans, selected)
	knownProviders := make(map[string]struct{}, len(providerPlans))
	moduleFile := scanCtx.primaryTerraformFile()

	for _, provider := range providerPlans {
		source := provider.update.ProviderSource()
		knownProviders[source] = struct{}{}

		selectedVersion, ok := selections[source]
		if !ok || selectedVersion.IsZero() {
			continue
		}

		tfFile := moduleFile
		if provider.required != nil {
			if providerFile := scanCtx.findProviderFile(provider.required.Name); providerFile != "" {
				tfFile = providerFile
			}
		}
		if tfFile == "" {
			continue
		}

		rootConstraint := ""
		if len(provider.baseConstraints) > 0 {
			rootConstraint = provider.baseConstraints[0]
		}
		if provider.required != nil && s.config.PinEnabled() {
			rootConstraint = selectedVersion.String()
		}

		plan.Providers = append(plan.Providers, domain.LockProviderSync{
			ProviderSource: source,
			Version:        selectedVersion.String(),
			Constraint:     versionkit.MergeConstraints(rootConstraint, mergedConstraints[source]),
			TerraformFile:  tfFile,
		})
	}

	for source, constraints := range mergedConstraints {
		if _, ok := knownProviders[source]; ok {
			continue
		}
		if moduleFile == "" {
			continue
		}

		version, err := s.resolveTransitiveProviderVersion(source, constraints)
		if err != nil {
			return domain.LockSyncPlan{}, err
		}

		plan.Providers = append(plan.Providers, domain.LockProviderSync{
			ProviderSource: source,
			Version:        version.String(),
			Constraint:     versionkit.MergeConstraints("", constraints),
			TerraformFile:  moduleFile,
		})
	}

	sort.Slice(plan.Providers, func(i, j int) bool {
		if plan.Providers[i].TerraformFile != plan.Providers[j].TerraformFile {
			return plan.Providers[i].TerraformFile < plan.Providers[j].TerraformFile
		}
		return plan.Providers[i].ProviderSource < plan.Providers[j].ProviderSource
	})
	return plan, nil
}

func (s *Solver) resolveTransitiveProviderVersion(source string, constraints []string) (versionkit.Version, error) {
	address, err := resolveRegistryProviderAddress(source, nil, s.config)
	if err != nil {
		return versionkit.Version{}, err
	}

	versionStrings, err := s.registry.ProviderVersions(s.ctx, address)
	if err != nil {
		return versionkit.Version{}, err
	}

	parsedConstraints, err := parseConstraintStrings(constraints)
	if err != nil {
		return versionkit.Version{}, err
	}

	selected, ok := selectProviderVersion(&providerPlan{
		update:   domain.NewProviderVersionUpdate(domain.ProviderDependency{ProviderSource: source}),
		address:  address,
		versions: parseVersionList(versionStrings),
	}, parsedConstraints, s.config.BumpPolicy())
	if !ok {
		return versionkit.Version{}, fmt.Errorf("no provider version satisfies merged transitive constraints for %s", source)
	}
	return selected, nil
}
