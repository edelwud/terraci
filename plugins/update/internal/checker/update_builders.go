package checker

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

func newModuleDependency(modulePath string, call *parser.ModuleCall) updateengine.ModuleDependency {
	return updateengine.ModuleDependency{
		ModulePath: modulePath,
		CallName:   call.Name,
		Source:     call.Source,
		Constraint: call.Version,
	}
}

func newModuleUpdate(dep updateengine.ModuleDependency) updateengine.ModuleVersionUpdate {
	return updateengine.NewModuleVersionUpdate(dep)
}

func skipModuleUpdate(
	update updateengine.ModuleVersionUpdate,
	issue string,
) updateengine.ModuleVersionUpdate {
	update.Status = updateengine.StatusSkipped
	update.Issue = issue
	return update
}

func errorModuleUpdate(
	update updateengine.ModuleVersionUpdate,
	err error,
) updateengine.ModuleVersionUpdate {
	update.Status = updateengine.StatusError
	update.Issue = fmt.Sprintf("registry error: %v", err)
	return update
}

func markModuleUpdateAvailable(
	update updateengine.ModuleVersionUpdate,
	file string,
	bumpedVersion string,
) updateengine.ModuleVersionUpdate {
	update.File = file
	update.BumpedVersion = bumpedVersion
	update.Status = updateengine.StatusUpdateAvailable
	return update
}

func newProviderDependency(
	modulePath string,
	requiredProvider *parser.RequiredProvider,
) updateengine.ProviderDependency {
	return updateengine.ProviderDependency{
		ModulePath:     modulePath,
		ProviderName:   requiredProvider.Name,
		ProviderSource: requiredProvider.Source,
		Constraint:     requiredProvider.VersionConstraint,
	}
}

func newProviderUpdate(dep updateengine.ProviderDependency) updateengine.ProviderVersionUpdate {
	return updateengine.NewProviderVersionUpdate(dep)
}

func skipProviderUpdate(
	update updateengine.ProviderVersionUpdate,
	issue string,
) updateengine.ProviderVersionUpdate {
	update.Status = updateengine.StatusSkipped
	update.Issue = issue
	return update
}

func errorProviderUpdate(
	update updateengine.ProviderVersionUpdate,
	err error,
) updateengine.ProviderVersionUpdate {
	update.Status = updateengine.StatusError
	update.Issue = fmt.Sprintf("registry error: %v", err)
	return update
}

func markProviderUpdateAvailable(
	update updateengine.ProviderVersionUpdate,
	file string,
	bumpedVersion string,
) updateengine.ProviderVersionUpdate {
	update.File = file
	update.BumpedVersion = bumpedVersion
	update.Status = updateengine.StatusUpdateAvailable
	return update
}
