package checker

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
	"github.com/edelwud/terraci/plugins/update/internal/registryclient"
)

func (s *checkSession) collectModuleUpdates(
	scanCtx *moduleScanContext,
) {
	for _, mc := range scanCtx.parsed.ModuleCalls {
		if mc.IsLocal || mc.Version == "" || !registryclient.IsRegistrySource(mc.Source) {
			continue
		}
		s.addModuleUpdate(scanCtx, mc)
	}
}

func (s *checkSession) addModuleUpdate(
	scanCtx *moduleScanContext,
	call *parser.ModuleCall,
) {
	s.builder.AddModuleUpdate(s.scanModuleCall(scanCtx, call))
}

func (s *checkSession) scanModuleCall(
	scanCtx *moduleScanContext,
	call *parser.ModuleCall,
) updateengine.ModuleVersionUpdate {
	dependency := newModuleDependency(scanCtx.module.RelativePath, call)
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

	analysis := analyzeModuleVersions(call.Version, parseVersionList(versionStrings), s.checker.config.Bump)
	if analysis.hasCurrent {
		update.CurrentVersion = analysis.current.String()
	}
	if !analysis.latest.IsZero() {
		update.LatestVersion = analysis.latest.String()
	}
	if !analysis.bumped.IsZero() {
		return markModuleUpdateAvailable(
			update,
			scanCtx.fileIndex.FindModuleBlockFile(call.Name),
			analysis.bumped.String(),
		)
	}

	return update
}
