package parser

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	dependencyengine "github.com/edelwud/terraci/pkg/parser/internal/dependency"
	"github.com/edelwud/terraci/pkg/parser/internal/model"
)

type dependencyParserAdapter struct {
	inner ModuleParser
}

func newDependencyParserAdapter(parser ModuleParser) dependencyParserAdapter {
	return dependencyParserAdapter{inner: parser}
}

func (a dependencyParserAdapter) ParseModule(ctx context.Context, modulePath string) (*model.ParsedModule, error) {
	parsed, err := a.inner.ParseModule(ctx, modulePath)
	if err != nil {
		return nil, err
	}

	return toInternalParsedModule(parsed), nil
}

func (a dependencyParserAdapter) ResolveWorkspacePath(
	ref *model.RemoteStateRef,
	modulePath string,
	locals, variables map[string]cty.Value,
) ([]string, error) {
	return a.inner.ResolveWorkspacePath(fromInternalRemoteState(ref), modulePath, locals, variables)
}

func toInternalParsedModule(parsed *ParsedModule) *model.ParsedModule {
	if parsed == nil {
		return nil
	}

	result := model.NewParsedModule(parsed.Path)
	result.Locals = parsed.Locals
	result.Variables = parsed.Variables
	result.Backend = toInternalBackend(parsed.Backend)
	result.RequiredProviders = toInternalRequiredProviders(parsed.RequiredProviders)
	result.LockedProviders = toInternalLockedProviders(parsed.LockedProviders)
	result.RemoteStates = toInternalRemoteStates(parsed.RemoteStates)
	result.ModuleCalls = toInternalModuleCalls(parsed.ModuleCalls)
	result.Files = parsed.Files
	result.Diagnostics = append(hcl.Diagnostics(nil), parsed.Diagnostics...)
	result.TopLevelBlocks = cloneTopLevelBlocks(parsed.topLevelBlocks)
	return result
}

func fromInternalParsedModule(parsed *model.ParsedModule) *ParsedModule {
	if parsed == nil {
		return nil
	}

	return &ParsedModule{
		Path:              parsed.Path,
		Locals:            parsed.Locals,
		Variables:         parsed.Variables,
		Backend:           fromInternalBackend(parsed.Backend),
		RequiredProviders: fromInternalRequiredProviders(parsed.RequiredProviders),
		LockedProviders:   fromInternalLockedProviders(parsed.LockedProviders),
		RemoteStates:      fromInternalRemoteStates(parsed.RemoteStates),
		ModuleCalls:       fromInternalModuleCalls(parsed.ModuleCalls),
		Files:             parsed.Files,
		Diagnostics:       append(hcl.Diagnostics(nil), parsed.Diagnostics...),
		topLevelBlocks:    cloneTopLevelBlocks(parsed.TopLevelBlocks),
	}
}

func toInternalBackend(backend *BackendConfig) *model.BackendConfig {
	if backend == nil {
		return nil
	}

	return &model.BackendConfig{
		Type:   backend.Type,
		Config: backend.Config,
	}
}

func fromInternalBackend(backend *model.BackendConfig) *BackendConfig {
	if backend == nil {
		return nil
	}

	return &BackendConfig{
		Type:   backend.Type,
		Config: backend.Config,
	}
}

func toInternalRequiredProviders(providers []*RequiredProvider) []*model.RequiredProvider {
	result := make([]*model.RequiredProvider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		result = append(result, &model.RequiredProvider{
			Name:              provider.Name,
			Source:            provider.Source,
			VersionConstraint: provider.VersionConstraint,
		})
	}
	return result
}

func fromInternalRequiredProviders(providers []*model.RequiredProvider) []*RequiredProvider {
	result := make([]*RequiredProvider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		result = append(result, &RequiredProvider{
			Name:              provider.Name,
			Source:            provider.Source,
			VersionConstraint: provider.VersionConstraint,
		})
	}
	return result
}

func toInternalLockedProviders(providers []*LockedProvider) []*model.LockedProvider {
	result := make([]*model.LockedProvider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		result = append(result, &model.LockedProvider{
			Source:      provider.Source,
			Version:     provider.Version,
			Constraints: provider.Constraints,
		})
	}
	return result
}

func fromInternalLockedProviders(providers []*model.LockedProvider) []*LockedProvider {
	result := make([]*LockedProvider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		result = append(result, &LockedProvider{
			Source:      provider.Source,
			Version:     provider.Version,
			Constraints: provider.Constraints,
		})
	}
	return result
}

func toInternalRemoteStates(remoteStates []*RemoteStateRef) []*model.RemoteStateRef {
	result := make([]*model.RemoteStateRef, 0, len(remoteStates))
	for _, remoteState := range remoteStates {
		if remoteState == nil {
			continue
		}
		result = append(result, &model.RemoteStateRef{
			Name:         remoteState.Name,
			Backend:      remoteState.Backend,
			Config:       remoteState.Config,
			ForEach:      remoteState.ForEach,
			WorkspaceDir: remoteState.WorkspaceDir,
			RawBody:      remoteState.RawBody,
		})
	}
	return result
}

func fromInternalRemoteStates(remoteStates []*model.RemoteStateRef) []*RemoteStateRef {
	result := make([]*RemoteStateRef, 0, len(remoteStates))
	for _, remoteState := range remoteStates {
		if remoteState == nil {
			continue
		}
		result = append(result, fromInternalRemoteState(remoteState))
	}
	return result
}

func fromInternalRemoteState(remoteState *model.RemoteStateRef) *RemoteStateRef {
	if remoteState == nil {
		return nil
	}

	return &RemoteStateRef{
		Name:         remoteState.Name,
		Backend:      remoteState.Backend,
		Config:       remoteState.Config,
		ForEach:      remoteState.ForEach,
		WorkspaceDir: remoteState.WorkspaceDir,
		RawBody:      remoteState.RawBody,
	}
}

func toInternalModuleCalls(moduleCalls []*ModuleCall) []*model.ModuleCall {
	result := make([]*model.ModuleCall, 0, len(moduleCalls))
	for _, moduleCall := range moduleCalls {
		if moduleCall == nil {
			continue
		}
		result = append(result, &model.ModuleCall{
			Name:         moduleCall.Name,
			Source:       moduleCall.Source,
			Version:      moduleCall.Version,
			IsLocal:      moduleCall.IsLocal,
			ResolvedPath: moduleCall.ResolvedPath,
		})
	}
	return result
}

func fromInternalModuleCalls(moduleCalls []*model.ModuleCall) []*ModuleCall {
	result := make([]*ModuleCall, 0, len(moduleCalls))
	for _, moduleCall := range moduleCalls {
		if moduleCall == nil {
			continue
		}
		result = append(result, &ModuleCall{
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

func fromInternalModuleCall(moduleCall *model.ModuleCall) *ModuleCall {
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

func cloneTopLevelBlocks(blocks map[string][]*hcl.Block) map[string][]*hcl.Block {
	cloned := make(map[string][]*hcl.Block, len(blocks))
	for blockType, entries := range blocks {
		cloned[blockType] = append([]*hcl.Block(nil), entries...)
	}

	return cloned
}
