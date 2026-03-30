package checker

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
	"github.com/edelwud/terraci/plugins/update/internal/registryclient"
)

func (s *checkSession) collectModuleUpdates(
	mod *discovery.Module,
	parsed *parser.ParsedModule,
) {
	for _, mc := range parsed.ModuleCalls {
		if mc.IsLocal || mc.Version == "" || !registryclient.IsRegistrySource(mc.Source) {
			continue
		}
		s.addModuleUpdate(mod, mc)
	}
}

func (s *checkSession) addModuleUpdate(
	mod *discovery.Module,
	call *parser.ModuleCall,
) {
	s.builder.AddModuleUpdate(s.scanModuleCall(mod, call))
}

func (s *checkSession) scanModuleCall(
	mod *discovery.Module,
	call *parser.ModuleCall,
) updateengine.ModuleVersionUpdate {
	dependency := newModuleDependency(mod.RelativePath, call)
	update := newModuleUpdate(dependency)

	if s.checker.config.IsIgnored(call.Source) {
		return skipModuleUpdate(update, skipReasonIgnored)
	}

	namespace, name, provider, err := registryclient.ParseModuleSource(call.Source)
	if err != nil {
		return skipModuleUpdate(update, fmt.Sprintf("invalid source: %v", err))
	}

	versionStrings, err := s.checker.registry.ModuleVersions(s.ctx, namespace, name, provider)
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

	bumped, ok := updateengine.LatestByBump(current, versions, s.checker.config.Bump)
	if ok {
		return markModuleUpdateAvailable(update, mod.Path, call.Name, bumped.String())
	}

	return update
}
