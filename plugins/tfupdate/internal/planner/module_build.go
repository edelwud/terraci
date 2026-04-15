package planner

import (
	"fmt"
	"sort"

	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/versionkit"
)

func (s *Solver) buildModulePlan(scanCtx *moduleScanContext, call *parser.ModuleCall) *modulePlan {
	update := domain.NewModuleVersionUpdate(domain.ModuleDependency{
		ModulePath: scanCtx.module.RelativePath,
		CallName:   call.Name,
		Source:     call.Source,
		Constraint: call.Version,
	})

	switch {
	case call.IsLocal:
		update.Status = domain.StatusSkipped
		update.Issue = "local module source is not supported"
		return &modulePlan{call: call, update: update}
	case call.Version == "":
		update.Status = domain.StatusSkipped
		update.Issue = "no version specified"
		return &modulePlan{call: call, update: update}
	case !sourceaddr.IsRegistryModuleSource(call.Source):
		update.Status = domain.StatusSkipped
		update.Issue = "non-registry module source is not supported"
		return &modulePlan{call: call, update: update}
	case s.config.IsIgnored(call.Source):
		update.Status = domain.StatusSkipped
		update.Issue = skipReasonIgnored
		return &modulePlan{call: call, update: update}
	}

	address, err := sourceaddr.ParseRegistryModuleSource(call.Source)
	if err != nil {
		update.Status = domain.StatusSkipped
		update.Issue = fmt.Sprintf("invalid source: %v", err)
		return &modulePlan{call: call, update: update}
	}
	address = address.WithHostname(s.config.DefaultRegistryHost())

	versionStrings, err := s.registry.ModuleVersions(s.ctx, address)
	if err != nil {
		update.Status = domain.StatusError
		update.Issue = fmt.Sprintf("registry error: %v", err)
		return &modulePlan{call: call, update: update}
	}

	parsedVersions := parseVersionList(versionStrings)
	current := versionFromConstraint(call.Version)
	latest := latestStable(parsedVersions)

	choices := s.buildModuleChoices(address, current, parsedVersions)
	updateChoices := countUpdateChoices(choices, current)

	return &modulePlan{call: call, update: update, address: address, current: current, latest: latest, choices: choices, updateChoices: updateChoices}
}

func (s *Solver) buildModuleChoices(address sourceaddr.ModuleAddress, current versionkit.Version, versions []versionkit.Version) []moduleChoice {
	var choices []moduleChoice
	seen := map[string]struct{}{}
	appendChoice := func(version versionkit.Version) {
		if version.IsZero() {
			return
		}
		key := version.String()
		if _, ok := seen[key]; ok {
			return
		}
		deps, err := s.registry.ModuleProviderDeps(s.ctx, address, key)
		if err != nil {
			return
		}
		seen[key] = struct{}{}
		choices = append(choices, moduleChoice{version: version, providerDeps: deps})
	}

	appendChoice(current)
	if s.config.ShouldCheckModules() {
		for _, version := range sortBumpCandidates(versions, current, s.config.BumpPolicy()) {
			appendChoice(version)
		}
	}
	sort.Slice(choices, func(i, j int) bool { return choices[i].version.Compare(choices[j].version) > 0 })
	return choices
}

func countUpdateChoices(choices []moduleChoice, current versionkit.Version) int {
	count := 0
	for _, choice := range choices {
		if choice.version.Compare(current) > 0 {
			count++
		}
	}
	return count
}
