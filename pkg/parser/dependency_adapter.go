package parser

import (
	"context"

	"github.com/zclconf/go-cty/cty"

	dependencyengine "github.com/edelwud/terraci/pkg/parser/internal/dependency"
)

type dependencyParserAdapter struct {
	inner ModuleParser
}

func newDependencyParserAdapter(parser ModuleParser) dependencyParserAdapter {
	return dependencyParserAdapter{inner: parser}
}

func (a dependencyParserAdapter) ParseModule(ctx context.Context, modulePath string) (*dependencyengine.ParsedModule, error) {
	parsed, err := a.inner.ParseModule(ctx, modulePath)
	if err != nil {
		return nil, err
	}

	return &dependencyengine.ParsedModule{
		Locals:       parsed.Locals,
		Variables:    parsed.Variables,
		Backend:      toDependencyBackend(parsed.Backend),
		RemoteStates: toDependencyRemoteStates(parsed.RemoteStates),
		ModuleCalls:  toDependencyModuleCalls(parsed.ModuleCalls),
	}, nil
}

func (a dependencyParserAdapter) ResolveWorkspacePath(
	ref *dependencyengine.RemoteStateRef,
	modulePath string,
	locals, variables map[string]cty.Value,
) ([]string, error) {
	return a.inner.ResolveWorkspacePath(&RemoteStateRef{
		Name:    ref.Name,
		Backend: ref.Backend,
		Config:  ref.Config,
		ForEach: ref.ForEach,
	}, modulePath, locals, variables)
}

func toDependencyBackend(backend *BackendConfig) *dependencyengine.BackendConfig {
	if backend == nil {
		return nil
	}

	return &dependencyengine.BackendConfig{
		Type:   backend.Type,
		Config: backend.Config,
	}
}

func toDependencyRemoteStates(remoteStates []*RemoteStateRef) []*dependencyengine.RemoteStateRef {
	result := make([]*dependencyengine.RemoteStateRef, 0, len(remoteStates))
	for _, remoteState := range remoteStates {
		if remoteState == nil {
			continue
		}

		result = append(result, &dependencyengine.RemoteStateRef{
			Name:    remoteState.Name,
			Backend: remoteState.Backend,
			Config:  remoteState.Config,
			ForEach: remoteState.ForEach,
		})
	}

	return result
}

func toDependencyModuleCalls(moduleCalls []*ModuleCall) []*dependencyengine.ModuleCall {
	result := make([]*dependencyengine.ModuleCall, 0, len(moduleCalls))
	for _, moduleCall := range moduleCalls {
		if moduleCall == nil {
			continue
		}

		result = append(result, &dependencyengine.ModuleCall{
			Name:         moduleCall.Name,
			Source:       moduleCall.Source,
			Version:      moduleCall.Version,
			IsLocal:      moduleCall.IsLocal,
			ResolvedPath: moduleCall.ResolvedPath,
		})
	}

	return result
}

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
			ModuleCall:  fromDependencyModuleCall(libraryDependency.ModuleCall),
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

func fromDependencyModuleCall(moduleCall *dependencyengine.ModuleCall) *ModuleCall {
	if moduleCall == nil {
		return nil
	}

	return &ModuleCall{
		Name:         moduleCall.Name,
		Source:       moduleCall.Source,
		Version:      moduleCall.Version,
		IsLocal:      moduleCall.IsLocal,
		ResolvedPath: moduleCall.ResolvedPath,
	}
}
