package checker

import (
	"context"
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
	"github.com/edelwud/terraci/plugins/update/internal/registryclient"
)

func (s *Checker) checkProviderUpdates(
	ctx context.Context,
	mod *discovery.Module,
	parsed *parser.ParsedModule,
	result *updateengine.UpdateResult,
) {
	lockIndex := buildLockIndex(parsed.LockedProviders)

	for _, rp := range parsed.RequiredProviders {
		s.appendProviderUpdate(ctx, mod, rp, lockIndex, result)
	}
}

func (s *Checker) appendProviderUpdate(
	ctx context.Context,
	mod *discovery.Module,
	requiredProvider *parser.RequiredProvider,
	lockIndex map[string]*parser.LockedProvider,
	result *updateengine.UpdateResult,
) {
	result.Providers = append(result.Providers, s.scanProvider(ctx, mod, requiredProvider, lockIndex))
}

func (s *Checker) scanProvider(
	ctx context.Context,
	mod *discovery.Module,
	requiredProvider *parser.RequiredProvider,
	lockIndex map[string]*parser.LockedProvider,
) updateengine.ProviderVersionUpdate {
	update := newProviderUpdate(mod.RelativePath, requiredProvider)

	switch {
	case requiredProvider.Source == "":
		return skipProviderUpdate(update, "no source specified")
	case s.config.IsIgnored(requiredProvider.Source):
		return skipProviderUpdate(update, skipReasonIgnored)
	}

	update = withLockedProviderState(update, lockIndex[requiredProvider.Source])

	namespace, typeName, err := registryclient.ParseProviderSource(requiredProvider.Source)
	if err != nil {
		return skipProviderUpdate(update, fmt.Sprintf("invalid source: %v", err))
	}

	versionStrings, err := s.registry.ProviderVersions(ctx, namespace, typeName)
	if err != nil {
		return errorProviderUpdate(update, err)
	}

	versions := parseVersionList(versionStrings)
	current := resolveProviderCurrentVersion(update.Constraint, update.CurrentVersion, versions)
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

	bumped, ok := updateengine.LatestByBump(current, versions, s.config.Bump)
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
	if update.Constraint == "" && lockedProvider.Constraints != "" {
		update.Constraint = lockedProvider.Constraints
	}
	return update
}
