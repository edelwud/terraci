package checker

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
	"github.com/edelwud/terraci/plugins/update/internal/tffile"
)

func newModuleUpdate(modulePath string, call *parser.ModuleCall) updateengine.ModuleVersionUpdate {
	return updateengine.ModuleVersionUpdate{
		ModulePath: modulePath,
		CallName:   call.Name,
		Source:     call.Source,
		Constraint: call.Version,
	}
}

func skipModuleUpdate(
	update updateengine.ModuleVersionUpdate,
	reason string,
) updateengine.ModuleVersionUpdate {
	update.Skipped = true
	update.SkipReason = reason
	return update
}

func errorModuleUpdate(
	update updateengine.ModuleVersionUpdate,
	err error,
) updateengine.ModuleVersionUpdate {
	update.Error = fmt.Sprintf("registry error: %v", err)
	return update
}

func markModuleUpdateAvailable(
	update updateengine.ModuleVersionUpdate,
	modulePath string,
	callName string,
	bumpedVersion string,
) updateengine.ModuleVersionUpdate {
	update.File = tffile.FindModuleBlockFile(modulePath, callName)
	update.BumpedVersion = bumpedVersion
	update.UpdateAvailable = true
	return update
}

func newProviderUpdate(
	modulePath string,
	requiredProvider *parser.RequiredProvider,
) updateengine.ProviderVersionUpdate {
	return updateengine.ProviderVersionUpdate{
		ModulePath:     modulePath,
		ProviderName:   requiredProvider.Name,
		ProviderSource: requiredProvider.Source,
		Constraint:     requiredProvider.VersionConstraint,
	}
}

func skipProviderUpdate(
	update updateengine.ProviderVersionUpdate,
	reason string,
) updateengine.ProviderVersionUpdate {
	update.Skipped = true
	update.SkipReason = reason
	return update
}

func errorProviderUpdate(
	update updateengine.ProviderVersionUpdate,
	err error,
) updateengine.ProviderVersionUpdate {
	update.Error = fmt.Sprintf("registry error: %v", err)
	return update
}

func markProviderUpdateAvailable(
	update updateengine.ProviderVersionUpdate,
	modulePath string,
	providerName string,
	bumpedVersion string,
) updateengine.ProviderVersionUpdate {
	update.File = tffile.FindProviderBlockFile(modulePath, providerName)
	update.BumpedVersion = bumpedVersion
	update.UpdateAvailable = true
	return update
}
