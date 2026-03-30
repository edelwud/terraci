package checker

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
	"github.com/edelwud/terraci/plugins/update/internal/registryclient"
)

func (s *checkSession) collectProviderUpdates(
	mod *discovery.Module,
	parsed *parser.ParsedModule,
) {
	lockIndex := buildLockIndex(parsed.LockedProviders)

	for _, rp := range parsed.RequiredProviders {
		s.addProviderUpdate(mod, rp, lockIndex)
	}
}

func (s *checkSession) addProviderUpdate(
	mod *discovery.Module,
	requiredProvider *parser.RequiredProvider,
	lockIndex map[string]*parser.LockedProvider,
) {
	s.builder.AddProviderUpdate(s.scanProvider(mod, requiredProvider, lockIndex))
}

func (s *checkSession) scanProvider(
	mod *discovery.Module,
	requiredProvider *parser.RequiredProvider,
	lockIndex map[string]*parser.LockedProvider,
) updateengine.ProviderVersionUpdate {
	dependency := newProviderDependency(mod.RelativePath, requiredProvider)
	update := newProviderUpdate(dependency)

	switch {
	case requiredProvider.Source == "":
		return skipProviderUpdate(update, "no source specified")
	case s.checker.config.IsIgnored(requiredProvider.Source):
		return skipProviderUpdate(update, skipReasonIgnored)
	}

	update = withLockedProviderState(update, lockIndex[requiredProvider.Source])

	namespace, typeName, err := registryclient.ParseProviderSource(requiredProvider.Source)
	if err != nil {
		return skipProviderUpdate(update, fmt.Sprintf("invalid source: %v", err))
	}

	versionStrings, err := s.checker.registry.ProviderVersions(s.ctx, namespace, typeName)
	if err != nil {
		return errorProviderUpdate(update, err)
	}

	versions := parseVersionList(versionStrings)
	current := resolveProviderCurrentVersion(update.Constraint(), update.CurrentVersion, versions)
	if !current.IsZero() {
		update.CurrentVersion = current.String()
	}

	latest := latestStable(versions)
	if !latest.IsZero() {
		update.LatestVersion = latest.String()
	}

	if current.IsZero() {
		return skipProviderUpdate(update, "cannot determine current version")
	}

	bumped, ok := updateengine.LatestByBump(current, versions, s.checker.config.Bump)
	if ok {
		return markProviderUpdateAvailable(update, mod.Path, requiredProvider.Name, bumped.String())
	}

	return update
}

func resolveProviderCurrentVersion(constraint, currentVersion string, versions []updateengine.Version) updateengine.Version {
	if currentVersion != "" {
		if version, err := updateengine.ParseVersion(currentVersion); err == nil {
			return version
		}
	}
	if constraint == "" {
		return updateengine.Version{}
	}
	constraints, err := updateengine.ParseConstraints(constraint)
	if err != nil {
		return updateengine.Version{}
	}
	resolved, found := updateengine.LatestAllowed(versions, constraints)
	if !found {
		return updateengine.Version{}
	}
	return resolved
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
