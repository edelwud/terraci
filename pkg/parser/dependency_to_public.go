package parser

import (
	dependencyengine "github.com/edelwud/terraci/pkg/parser/internal/dependency"
)

func fromDependencyModuleDependencies(deps *dependencyengine.ModuleDependencies) *ModuleDependencies {
	if deps == nil {
		return nil
	}

	result := &ModuleDependencies{
		Module:              deps.Module,
		Dependencies:        make([]*Dependency, 0, len(deps.Dependencies)),
		LibraryDependencies: make([]*LibraryDependency, 0, len(deps.LibraryDependencies)),
		DependsOn:           append([]string(nil), deps.DependsOn...),
		Errors:              append([]error(nil), deps.Errors...),
	}

	for _, dependency := range deps.Dependencies {
		if dependency == nil {
			continue
		}

		result.Dependencies = append(result.Dependencies, &Dependency{
			From:            dependency.From,
			To:              dependency.To,
			Type:            dependency.Type,
			RemoteStateName: dependency.RemoteStateName,
		})
	}

	for _, libraryDependency := range deps.LibraryDependencies {
		if libraryDependency == nil {
			continue
		}

		result.LibraryDependencies = append(result.LibraryDependencies, &LibraryDependency{
			ModuleCall:  fromInternalModuleCall(libraryDependency.ModuleCall),
			LibraryPath: libraryDependency.LibraryPath,
		})
	}

	return result
}

func fromDependencyModuleDependenciesMap(results map[string]*dependencyengine.ModuleDependencies) map[string]*ModuleDependencies {
	converted := make(map[string]*ModuleDependencies, len(results))
	for moduleID, deps := range results {
		converted[moduleID] = fromDependencyModuleDependencies(deps)
	}

	return converted
}
