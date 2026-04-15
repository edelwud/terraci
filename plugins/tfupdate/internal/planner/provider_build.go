package planner

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

func (s *Solver) buildProviderPlan(scanCtx *moduleScanContext, required *parser.RequiredProvider) *providerPlan {
	update := domain.NewProviderVersionUpdate(domain.ProviderDependency{
		ModulePath:     scanCtx.module.RelativePath,
		ProviderName:   required.Name,
		ProviderSource: required.Source,
		Constraint:     required.VersionConstraint,
	})

	switch {
	case required.Source == "":
		update.Status = domain.StatusSkipped
		update.Issue = "no source specified"
		return &providerPlan{required: required, update: update}
	case s.config.IsIgnored(required.Source):
		update.Status = domain.StatusSkipped
		update.Issue = skipReasonIgnored
		return &providerPlan{required: required, update: update}
	}

	update = withLockedProviderState(update, scanCtx.lockIndex[required.Source])
	address, err := resolveRegistryProviderAddress(required.Source, scanCtx.lockIndex[required.Source], s.config)
	if err != nil {
		update.Status = domain.StatusSkipped
		update.Issue = fmt.Sprintf("invalid source: %v", err)
		return &providerPlan{required: required, update: update}
	}

	versionStrings, err := s.registry.ProviderVersions(s.ctx, address)
	if err != nil {
		update.Status = domain.StatusError
		update.Issue = fmt.Sprintf("registry error: %v", err)
		return &providerPlan{required: required, update: update}
	}

	versions := parseVersionList(versionStrings)
	analysis := analyzeProviderVersions(update.Constraint(), inferredCurrentVersion(update), versions, s.config.BumpPolicy())

	return &providerPlan{
		required:        required,
		update:          update,
		address:         address,
		current:         analysis.current,
		hasCurrent:      analysis.hasCurrent,
		latest:          analysis.latest,
		versions:        versions,
		baseConstraints: baseProviderConstraints(update),
		locked:          scanCtx.lockIndex[required.Source],
	}
}

func (s *Solver) buildLockedProviderPlan(scanCtx *moduleScanContext, locked *parser.LockedProvider) *providerPlan {
	if locked == nil || locked.Source == "" {
		return nil
	}

	address, err := sourceaddr.ParseProviderAddress(locked.Source)
	if err != nil {
		update := domain.NewProviderVersionUpdate(domain.ProviderDependency{
			ModulePath:     scanCtx.module.RelativePath,
			ProviderName:   "",
			ProviderSource: stripRegistryPrefix(locked.Source),
			Constraint:     locked.Constraints,
		})
		update.Status = domain.StatusSkipped
		update.Issue = fmt.Sprintf("invalid locked source: %v", err)
		return &providerPlan{update: update, locked: locked}
	}

	shortSource := address.ShortSource()
	update := domain.NewProviderVersionUpdate(domain.ProviderDependency{
		ModulePath:     scanCtx.module.RelativePath,
		ProviderName:   address.Type,
		ProviderSource: shortSource,
		Constraint:     locked.Constraints,
	})
	update.CurrentVersion = locked.Version

	versionStrings, err := s.registry.ProviderVersions(s.ctx, address)
	if err != nil {
		update.Status = domain.StatusError
		update.Issue = fmt.Sprintf("registry error: %v", err)
		return &providerPlan{update: update, address: address, locked: locked}
	}

	versions := parseVersionList(versionStrings)
	analysis := analyzeProviderVersions(update.Constraint(), update.CurrentVersion, versions, s.config.BumpPolicy())

	return &providerPlan{
		update:          update,
		address:         address,
		current:         analysis.current,
		hasCurrent:      analysis.hasCurrent,
		latest:          analysis.latest,
		versions:        versions,
		baseConstraints: baseProviderConstraints(update),
		locked:          locked,
	}
}

func inferredCurrentVersion(update domain.ProviderVersionUpdate) string {
	if update.CurrentVersion != "" || update.Constraint() == "" {
		return update.CurrentVersion
	}
	base := versionFromConstraint(update.Constraint())
	if base.IsZero() {
		return ""
	}
	return base.String()
}

func baseProviderConstraints(update domain.ProviderVersionUpdate) []string {
	if update.Constraint() == "" {
		return nil
	}
	return []string{update.Constraint()}
}
