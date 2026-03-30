package checker

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
	"github.com/edelwud/terraci/plugins/update/internal/registryclient"
)

func (s *Checker) checkModuleUpdates(
	ctx context.Context,
	mod *discovery.Module,
	parsed *parser.ParsedModule,
	result *updateengine.UpdateResult,
) {
	for _, mc := range parsed.ModuleCalls {
		if mc.IsLocal || mc.Version == "" || !registryclient.IsRegistrySource(mc.Source) {
			continue
		}
		s.appendModuleUpdate(ctx, mod, mc, result)
	}
}

func (s *Checker) appendModuleUpdate(
	ctx context.Context,
	mod *discovery.Module,
	call *parser.ModuleCall,
	result *updateengine.UpdateResult,
) {
	result.Modules = append(result.Modules, s.scanModuleCall(ctx, mod, call))
}

func (s *Checker) scanModuleCall(
	ctx context.Context,
	mod *discovery.Module,
	call *parser.ModuleCall,
) updateengine.ModuleVersionUpdate {
	update := newModuleUpdate(mod.RelativePath, call)

	if s.config.IsIgnored(call.Source) {
		return skipModuleUpdate(update, skipReasonIgnored)
	}

	namespace, name, provider, err := registryclient.ParseModuleSource(call.Source)
	if err != nil {
		return skipModuleUpdate(update, fmt.Sprintf("invalid source: %v", err))
	}

	versionStrings, err := s.registry.ModuleVersions(ctx, namespace, name, provider)
	if err != nil {
		return errorModuleUpdate(update, err)
	}

	versions := parseVersionList(versionStrings)
	current := versionFromConstraint(call.Version)
	if !current.IsZero() {
		update.CurrentVersion = current.String()
	}

	latest := latestStable(versions)
	if !latest.IsZero() {
		update.LatestVersion = latest.String()
	}

	bumped, ok := updateengine.LatestByBump(current, versions, s.config.Bump)
	if ok {
		return markModuleUpdateAvailable(update, mod.Path, call.Name, bumped.String())
	}

	return update
}
