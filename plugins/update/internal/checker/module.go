package checker

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
	"github.com/edelwud/terraci/plugins/update/internal/registryclient"
)

func (s *checkSession) collectModuleUpdates(
	scanCtx *moduleScanContext,
) []updateengine.ModuleVersionUpdate {
	updates := make([]updateengine.ModuleVersionUpdate, 0, len(scanCtx.parsed.ModuleCalls))
	for _, mc := range scanCtx.parsed.ModuleCalls {
		if mc.IsLocal || mc.Version == "" || !registryclient.IsRegistrySource(mc.Source) {
			continue
		}
		updates = append(updates, s.scanModuleCall(scanCtx, mc))
	}

	return updates
}

func (s *checkSession) scanModuleCall(
	scanCtx *moduleScanContext,
	call *parser.ModuleCall,
) updateengine.ModuleVersionUpdate {
	update := newModuleUpdate(newModuleDependency(scanCtx.module.RelativePath, call))

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

	result := newModuleScanResult(
		update,
		analyzeModuleVersions(call.Version, parseVersionList(versionStrings), s.checker.config.Bump),
	)
	return result.outcome(func() string {
		return scanCtx.findModuleFile(call.Name)
	})
}
