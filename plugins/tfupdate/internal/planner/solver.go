package planner

import (
	"fmt"
	"maps"
	"sort"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

type moduleChoice struct {
	version      tfupdateengine.Version
	providerDeps []registrymeta.ModuleProviderDep
}

type modulePlan struct {
	call          *parser.ModuleCall
	update        tfupdateengine.ModuleVersionUpdate
	address       sourceaddr.ModuleAddress
	current       tfupdateengine.Version
	latest        tfupdateengine.Version
	choices       []moduleChoice
	updateChoices int
}

type providerPlan struct {
	required        *parser.RequiredProvider
	update          tfupdateengine.ProviderVersionUpdate
	address         sourceaddr.ProviderAddress
	current         tfupdateengine.Version
	hasCurrent      bool
	latest          tfupdateengine.Version
	versions        []tfupdateengine.Version
	baseConstraints []string
	locked          *parser.LockedProvider
}

type selectedModule struct {
	plan   *modulePlan
	choice moduleChoice
}

func (s *Solver) SolveModule(mod *discovery.Module, parsed *parser.ParsedModule) (*tfupdateengine.ModulePlan, error) {
	scanCtx := newModuleScanContext(mod, parsed)
	modulePlans, providerPlans := s.buildPlans(scanCtx)
	selected, providerSelections, compatible := s.findCompatiblePlan(modulePlans, providerPlans)
	lockSync, err := s.buildLockSyncPlan(scanCtx, providerPlans, selected, providerSelections, compatible)
	if err != nil {
		return nil, err
	}

	return &tfupdateengine.ModulePlan{
		ModulePath: mod.RelativePath,
		Modules:    s.materializeModuleResolutions(scanCtx, modulePlans, selected, compatible),
		Providers:  s.materializeProviderResolutions(scanCtx, providerPlans, providerSelections, compatible),
		LockSync:   lockSync,
	}, nil
}

func (s *Solver) buildPlans(scanCtx *moduleScanContext) ([]*modulePlan, []*providerPlan) {
	modulePlans := make([]*modulePlan, 0, len(scanCtx.parsed.ModuleCalls))
	for _, mc := range scanCtx.parsed.ModuleCalls {
		modulePlans = append(modulePlans, s.buildModulePlan(scanCtx, mc))
	}

	providerPlans := make([]*providerPlan, 0, len(scanCtx.parsed.RequiredProviders)+len(scanCtx.parsed.LockedProviders))
	seenProviders := make(map[string]struct{}, len(scanCtx.parsed.RequiredProviders))
	for _, rp := range scanCtx.parsed.RequiredProviders {
		providerPlans = append(providerPlans, s.buildProviderPlan(scanCtx, rp))
		seenProviders[rp.Source] = struct{}{}
	}
	for _, lp := range scanCtx.parsed.LockedProviders {
		short := stripRegistryPrefix(lp.Source)
		if _, ok := seenProviders[short]; ok {
			continue
		}
		if plan := s.buildLockedProviderPlan(scanCtx, lp); plan != nil {
			providerPlans = append(providerPlans, plan)
		}
	}
	return modulePlans, providerPlans
}

func (s *Solver) buildModulePlan(scanCtx *moduleScanContext, call *parser.ModuleCall) *modulePlan {
	update := tfupdateengine.NewModuleVersionUpdate(tfupdateengine.ModuleDependency{
		ModulePath: scanCtx.module.RelativePath,
		CallName:   call.Name,
		Source:     call.Source,
		Constraint: call.Version,
	})

	switch {
	case call.IsLocal:
		update.Status = tfupdateengine.StatusSkipped
		update.Issue = "local module source is not supported"
		return &modulePlan{call: call, update: update}
	case call.Version == "":
		update.Status = tfupdateengine.StatusSkipped
		update.Issue = "no version specified"
		return &modulePlan{call: call, update: update}
	case !sourceaddr.IsRegistryModuleSource(call.Source):
		update.Status = tfupdateengine.StatusSkipped
		update.Issue = "non-registry module source is not supported"
		return &modulePlan{call: call, update: update}
	case s.config.IsIgnored(call.Source):
		update.Status = tfupdateengine.StatusSkipped
		update.Issue = skipReasonIgnored
		return &modulePlan{call: call, update: update}
	}

	address, err := sourceaddr.ParseRegistryModuleSource(call.Source)
	if err != nil {
		update.Status = tfupdateengine.StatusSkipped
		update.Issue = fmt.Sprintf("invalid source: %v", err)
		return &modulePlan{call: call, update: update}
	}
	address = address.WithHostname(s.config.DefaultRegistryHost())

	versionStrings, err := s.registry.ModuleVersions(s.ctx, address.Hostname, address.Namespace, address.Name, address.Provider)
	if err != nil {
		update.Status = tfupdateengine.StatusError
		update.Issue = fmt.Sprintf("registry error: %v", err)
		return &modulePlan{call: call, update: update}
	}

	parsedVersions := parseVersionList(versionStrings)
	current := versionFromConstraint(call.Version)
	latest := latestStable(parsedVersions)

	var choices []moduleChoice
	seen := map[string]struct{}{}
	appendChoice := func(version tfupdateengine.Version) {
		if version.IsZero() {
			return
		}
		key := version.String()
		if _, ok := seen[key]; ok {
			return
		}
		deps, err := s.registry.ModuleProviderDeps(s.ctx, address.Hostname, address.Namespace, address.Name, address.Provider, key)
		if err != nil {
			return
		}
		seen[key] = struct{}{}
		choices = append(choices, moduleChoice{version: version, providerDeps: deps})
	}

	appendChoice(current)
	for _, version := range sortBumpCandidates(parsedVersions, current, s.config.BumpPolicy()) {
		appendChoice(version)
	}
	sort.Slice(choices, func(i, j int) bool { return choices[i].version.Compare(choices[j].version) > 0 })

	updateChoices := 0
	for _, choice := range choices {
		if choice.version.Compare(current) > 0 {
			updateChoices++
		}
	}
	return &modulePlan{call: call, update: update, address: address, current: current, latest: latest, choices: choices, updateChoices: updateChoices}
}

func (s *Solver) buildProviderPlan(scanCtx *moduleScanContext, required *parser.RequiredProvider) *providerPlan {
	update := tfupdateengine.NewProviderVersionUpdate(tfupdateengine.ProviderDependency{
		ModulePath:     scanCtx.module.RelativePath,
		ProviderName:   required.Name,
		ProviderSource: required.Source,
		Constraint:     required.VersionConstraint,
	})

	switch {
	case required.Source == "":
		update.Status = tfupdateengine.StatusSkipped
		update.Issue = "no source specified"
		return &providerPlan{required: required, update: update}
	case s.config.IsIgnored(required.Source):
		update.Status = tfupdateengine.StatusSkipped
		update.Issue = skipReasonIgnored
		return &providerPlan{required: required, update: update}
	}

	update = withLockedProviderState(update, scanCtx.lockIndex[required.Source])
	address, err := resolveRegistryProviderAddress(required.Source, scanCtx.lockIndex[required.Source], s.config)
	if err != nil {
		update.Status = tfupdateengine.StatusSkipped
		update.Issue = fmt.Sprintf("invalid source: %v", err)
		return &providerPlan{required: required, update: update}
	}

	versionStrings, err := s.registry.ProviderVersions(s.ctx, address.Hostname, address.Namespace, address.Type)
	if err != nil {
		update.Status = tfupdateengine.StatusError
		update.Issue = fmt.Sprintf("registry error: %v", err)
		return &providerPlan{required: required, update: update}
	}

	currentVersion := update.CurrentVersion
	if currentVersion == "" && update.Constraint() != "" {
		base := versionFromConstraint(update.Constraint())
		if !base.IsZero() {
			currentVersion = base.String()
		}
	}
	analysis := analyzeProviderVersions(update.Constraint(), currentVersion, parseVersionList(versionStrings), s.config.BumpPolicy())

	baseConstraints := make([]string, 0, 1)
	if update.Constraint() != "" {
		baseConstraints = append(baseConstraints, update.Constraint())
	}
	return &providerPlan{
		required:        required,
		update:          update,
		address:         address,
		current:         analysis.current,
		hasCurrent:      analysis.hasCurrent,
		latest:          analysis.latest,
		versions:        parseVersionList(versionStrings),
		baseConstraints: baseConstraints,
		locked:          scanCtx.lockIndex[required.Source],
	}
}

func (s *Solver) buildLockedProviderPlan(scanCtx *moduleScanContext, locked *parser.LockedProvider) *providerPlan {
	if locked == nil || locked.Source == "" {
		return nil
	}

	address, err := sourceaddr.ParseProviderAddress(locked.Source)
	if err != nil {
		update := tfupdateengine.NewProviderVersionUpdate(tfupdateengine.ProviderDependency{
			ModulePath:     scanCtx.module.RelativePath,
			ProviderName:   "",
			ProviderSource: stripRegistryPrefix(locked.Source),
			Constraint:     locked.Constraints,
		})
		update.Status = tfupdateengine.StatusSkipped
		update.Issue = fmt.Sprintf("invalid locked source: %v", err)
		return &providerPlan{update: update, locked: locked}
	}

	shortSource := address.ShortSource()
	update := tfupdateengine.NewProviderVersionUpdate(tfupdateengine.ProviderDependency{
		ModulePath:     scanCtx.module.RelativePath,
		ProviderName:   address.Type,
		ProviderSource: shortSource,
		Constraint:     locked.Constraints,
	})
	update.CurrentVersion = locked.Version

	versionStrings, err := s.registry.ProviderVersions(s.ctx, address.Hostname, address.Namespace, address.Type)
	if err != nil {
		update.Status = tfupdateengine.StatusError
		update.Issue = fmt.Sprintf("registry error: %v", err)
		return &providerPlan{update: update, address: address, locked: locked}
	}

	analysis := analyzeProviderVersions(update.Constraint(), update.CurrentVersion, parseVersionList(versionStrings), s.config.BumpPolicy())

	baseConstraints := make([]string, 0, 1)
	if update.Constraint() != "" {
		baseConstraints = append(baseConstraints, update.Constraint())
	}
	return &providerPlan{
		update:          update,
		address:         address,
		current:         analysis.current,
		hasCurrent:      analysis.hasCurrent,
		latest:          analysis.latest,
		versions:        parseVersionList(versionStrings),
		baseConstraints: baseConstraints,
		locked:          locked,
	}
}

func (s *Solver) findCompatiblePlan(modulePlans []*modulePlan, providerPlans []*providerPlan) (selectedModules map[string]selectedModule, providerVersions map[string]tfupdateengine.Version, ok bool) {
	selected := make(map[string]selectedModule, len(modulePlans))
	var search func(int) (map[string]selectedModule, map[string]tfupdateengine.Version, bool)
	search = func(idx int) (map[string]selectedModule, map[string]tfupdateengine.Version, bool) {
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
			constraints, err := tfupdateengine.ParseConstraints(dep.Version)
			if err != nil {
				continue
			}
			if !tfupdateengine.SatisfiesAll(plan.current, constraints) && len(plan.versions) == 0 {
				return false
			}
		}
	}
	return true
}

func solveProviders(providerPlans []*providerPlan, selected map[string]selectedModule, bump string) (map[string]tfupdateengine.Version, bool) {
	result := make(map[string]tfupdateengine.Version, len(providerPlans))
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

func selectProviderVersion(plan *providerPlan, constraints []tfupdateengine.Constraint, bump string) (tfupdateengine.Version, bool) {
	candidates := sortVersionsDesc(plan.versions)
	for _, version := range candidates {
		if version.Prerelease != "" {
			continue
		}
		if (!plan.hasCurrent || withinBump(plan.current, version, bump)) && tfupdateengine.SatisfiesAll(version, constraints) {
			return version, true
		}
	}
	if plan.hasCurrent && tfupdateengine.SatisfiesAll(plan.current, constraints) {
		return plan.current, true
	}
	return tfupdateengine.Version{}, false
}

func (s *Solver) materializeModuleResolutions(scanCtx *moduleScanContext, modulePlans []*modulePlan, selected map[string]selectedModule, compatible bool) []tfupdateengine.ModuleResolution {
	resolutions := make([]tfupdateengine.ModuleResolution, 0, len(modulePlans))
	for _, plan := range modulePlans {
		resolution := tfupdateengine.ModuleResolution{
			Dependency: plan.update.Dependency,
			Registry: tfupdateengine.RegistrySelection{
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
		if plan.update.Status == tfupdateengine.StatusError || plan.update.Status == tfupdateengine.StatusSkipped {
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
				resolution.Status = tfupdateengine.StatusUpdateAvailable
			} else {
				resolution.Status = tfupdateengine.StatusUpToDate
			}
		}
		if !compatible && plan.updateChoices > 0 {
			resolution.Status = tfupdateengine.StatusSkipped
			resolution.Issue = "no compatible provider resolution for module update policy"
		}
		resolutions = append(resolutions, resolution)
	}
	return resolutions
}

func (s *Solver) materializeProviderResolutions(scanCtx *moduleScanContext, providerPlans []*providerPlan, selections map[string]tfupdateengine.Version, compatible bool) []tfupdateengine.ProviderResolution {
	resolutions := make([]tfupdateengine.ProviderResolution, 0, len(providerPlans))
	for _, plan := range providerPlans {
		file := ""
		if plan.required != nil {
			file = scanCtx.findProviderFile(plan.required.Name)
		}
		resolution := tfupdateengine.ProviderResolution{
			Dependency: plan.update.Dependency,
			Registry: tfupdateengine.RegistrySelection{
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
		case plan.update.Status == tfupdateengine.StatusSkipped || plan.update.Status == tfupdateengine.StatusError:
			resolution.Status = plan.update.Status
			resolution.Issue = plan.update.Issue
		case !compatible || !ok:
			resolution.Status = tfupdateengine.StatusSkipped
			resolution.Issue = "no provider version satisfies merged direct and transitive constraints"
		case plan.hasCurrent && selected.Compare(plan.current) > 0:
			resolution.Selected = selected.String()
			resolution.Status = tfupdateengine.StatusUpdateAvailable
		case !plan.hasCurrent:
			resolution.Status = tfupdateengine.StatusSkipped
			resolution.Issue = "cannot determine current version"
		default:
			resolution.Status = tfupdateengine.StatusUpToDate
		}
		resolutions = append(resolutions, resolution)
	}
	return resolutions
}

func (s *Solver) buildLockSyncPlan(
	scanCtx *moduleScanContext,
	providerPlans []*providerPlan,
	selected map[string]selectedModule,
	selections map[string]tfupdateengine.Version,
	compatible bool,
) (tfupdateengine.LockSyncPlan, error) {
	plan := tfupdateengine.LockSyncPlan{ModulePath: scanCtx.module.RelativePath}
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

		plan.Providers = append(plan.Providers, tfupdateengine.LockProviderSync{
			ProviderSource: source,
			Version:        selectedVersion.String(),
			Constraint:     tfupdateengine.MergeConstraints(rootConstraint, mergedConstraints[source]),
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
			return tfupdateengine.LockSyncPlan{}, err
		}

		plan.Providers = append(plan.Providers, tfupdateengine.LockProviderSync{
			ProviderSource: source,
			Version:        version.String(),
			Constraint:     tfupdateengine.MergeConstraints("", constraints),
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

func (s *Solver) resolveTransitiveProviderVersion(source string, constraints []string) (tfupdateengine.Version, error) {
	address, err := resolveRegistryProviderAddress(source, nil, s.config)
	if err != nil {
		return tfupdateengine.Version{}, err
	}

	versionStrings, err := s.registry.ProviderVersions(s.ctx, address.Hostname, address.Namespace, address.Type)
	if err != nil {
		return tfupdateengine.Version{}, err
	}

	parsedConstraints, err := parseConstraintStrings(constraints)
	if err != nil {
		return tfupdateengine.Version{}, err
	}

	selected, ok := selectProviderVersion(&providerPlan{
		update:   tfupdateengine.NewProviderVersionUpdate(tfupdateengine.ProviderDependency{ProviderSource: source}),
		address:  address,
		versions: parseVersionList(versionStrings),
	}, parsedConstraints, s.config.BumpPolicy())
	if !ok {
		return tfupdateengine.Version{}, fmt.Errorf("no provider version satisfies merged transitive constraints for %s", source)
	}
	return selected, nil
}

func parseConstraintStrings(values []string) ([]tfupdateengine.Constraint, error) {
	var result []tfupdateengine.Constraint
	for _, value := range values {
		parsed, err := tfupdateengine.ParseConstraints(value)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed...)
	}
	return result, nil
}

func resolveRegistryProviderAddress(source string, lockedProvider *parser.LockedProvider, cfg *tfupdateengine.UpdateConfig) (sourceaddr.ProviderAddress, error) {
	address, err := sourceaddr.ParseProviderAddress(source)
	if err != nil {
		return sourceaddr.ProviderAddress{}, err
	}
	address = address.WithHostname(cfg.ProviderRegistryHost(address.ShortSource()))
	namespace, typeName, parseErr := sourceaddr.ParseProviderSource(source)
	if parseErr != nil || lockedProvider == nil || lockedProvider.Source == "" {
		return address, nil //nolint:nilerr // parse failure means short source is unusable; fall back to default address
	}
	lockedAddress, lockedErr := sourceaddr.ParseProviderAddress(lockedProvider.Source)
	if lockedErr != nil {
		return address, nil //nolint:nilerr // locked source parse failure is non-fatal; fall back to resolved address
	}
	if lockedAddress.Namespace != namespace || lockedAddress.Type != typeName {
		return address, nil
	}
	return lockedAddress, nil
}

func withLockedProviderState(update tfupdateengine.ProviderVersionUpdate, lockedProvider *parser.LockedProvider) tfupdateengine.ProviderVersionUpdate {
	if lockedProvider == nil {
		return update
	}
	if lockedProvider.Version != "" {
		update.CurrentVersion = lockedProvider.Version
	}
	if update.Constraint() == "" && lockedProvider.Constraints != "" {
		update.Dependency.Constraint = lockedProvider.Constraints
	}
	return update
}
