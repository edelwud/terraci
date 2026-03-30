package checker

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
	"github.com/edelwud/terraci/plugins/update/internal/registryclient"
)

func (s *checkSession) collectProviderUpdates(
	scanCtx *moduleScanContext,
) {
	for _, rp := range scanCtx.parsed.RequiredProviders {
		s.addProviderUpdate(scanCtx, rp)
	}
}

func (s *checkSession) addProviderUpdate(
	scanCtx *moduleScanContext,
	requiredProvider *parser.RequiredProvider,
) {
	s.builder.AddProviderUpdate(s.scanProvider(scanCtx, requiredProvider))
}

func (s *checkSession) scanProvider(
	scanCtx *moduleScanContext,
	requiredProvider *parser.RequiredProvider,
) updateengine.ProviderVersionUpdate {
	dependency := newProviderDependency(scanCtx.module.RelativePath, requiredProvider)
	update := newProviderUpdate(dependency)

	switch {
	case requiredProvider.Source == "":
		return skipProviderUpdate(update, "no source specified")
	case s.checker.config.IsIgnored(requiredProvider.Source):
		return skipProviderUpdate(update, skipReasonIgnored)
	}

	update = withLockedProviderState(update, scanCtx.lockIndex[requiredProvider.Source])

	namespace, typeName, err := registryclient.ParseProviderSource(requiredProvider.Source)
	if err != nil {
		return skipProviderUpdate(update, fmt.Sprintf("invalid source: %v", err))
	}

	versionStrings, err := s.checker.registry.ProviderVersions(s.ctx, namespace, typeName)
	if err != nil {
		return errorProviderUpdate(update, err)
	}

	analysis := analyzeProviderVersions(
		update.Constraint(),
		update.CurrentVersion,
		parseVersionList(versionStrings),
		s.checker.config.Bump,
	)
	if analysis.hasCurrent {
		update.CurrentVersion = analysis.current.String()
	}
	if !analysis.latest.IsZero() {
		update.LatestVersion = analysis.latest.String()
	}
	if !analysis.hasCurrent {
		return skipProviderUpdate(update, "cannot determine current version")
	}
	if !analysis.bumped.IsZero() {
		return markProviderUpdateAvailable(
			update,
			scanCtx.fileIndex.FindProviderBlockFile(requiredProvider.Name),
			analysis.bumped.String(),
		)
	}

	return update
}

// buildLockIndex creates a map from short provider source ("hashicorp/aws")
// to LockedProvider, stripping the registry hostname prefix.
func buildLockIndex(locked []*parser.LockedProvider) map[string]*parser.LockedProvider {
	idx := make(map[string]*parser.LockedProvider, len(locked))
	for _, lp := range locked {
		short := stripRegistryPrefix(lp.Source)
		idx[short] = lp
	}
	return idx
}

// stripRegistryPrefix removes the registry hostname prefix from a lock file source.
// Handles registry.terraform.io/ and registry.opentofu.org/.
func stripRegistryPrefix(source string) string {
	for _, prefix := range []string{"registry.terraform.io/", "registry.opentofu.org/"} {
		if strings.HasPrefix(source, prefix) {
			return source[len(prefix):]
		}
	}
	return source
}

func withLockedProviderState(
	update updateengine.ProviderVersionUpdate,
	lockedProvider *parser.LockedProvider,
) updateengine.ProviderVersionUpdate {
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
