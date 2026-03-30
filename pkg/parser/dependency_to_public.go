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
		if converted := fromDependencyEdge(dependency); converted != nil {
			result.Dependencies = append(result.Dependencies, converted)
		}
	}

	for _, libraryDependency := range deps.LibraryDependencies {
		if converted := fromLibraryDependency(libraryDependency); converted != nil {
			result.LibraryDependencies = append(result.LibraryDependencies, converted)
		}
	}

	return result
}

func fromDependencyEdge(dependency *dependencyengine.Dependency) *Dependency {
	if dependency == nil {
		return nil
	}

	return &Dependency{
		From:            dependency.From,
		To:              dependency.To,
		Type:            dependency.Type,
		RemoteStateName: dependency.RemoteStateName,
	}
}

func fromLibraryDependency(libraryDependency *dependencyengine.LibraryDependency) *LibraryDependency {
	if libraryDependency == nil {
		return nil
	}

	return &LibraryDependency{
		ModuleCall:  fromInternalModuleCall(libraryDependency.ModuleCall),
		LibraryPath: libraryDependency.LibraryPath,
	}
}

func fromDependencyModuleDependenciesMap(results map[string]*dependencyengine.ModuleDependencies) map[string]*ModuleDependencies {
	converted := make(map[string]*ModuleDependencies, len(results))
	for moduleID, deps := range results {
		converted[moduleID] = fromDependencyModuleDependencies(deps)
	}

	return converted
}
