package parser

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

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

func toInternalBackend(backend *BackendConfig) *model.BackendConfig {
	if backend == nil {
		return nil
	}

	return &model.BackendConfig{
		Type:   backend.Type,
		Config: backend.Config,
	}
}

func toInternalRequiredProviders(providers []*RequiredProvider) []*model.RequiredProvider {
	result := make([]*model.RequiredProvider, 0, len(providers))
	for _, provider := range providers {
		if converted := toInternalRequiredProvider(provider); converted != nil {
			result = append(result, converted)
		}
	}
	return result
}

func toInternalRequiredProvider(provider *RequiredProvider) *model.RequiredProvider {
	if provider == nil {
		return nil
	}

	return &model.RequiredProvider{
		Name:              provider.Name,
		Source:            provider.Source,
		VersionConstraint: provider.VersionConstraint,
	}
}

func toInternalLockedProviders(providers []*LockedProvider) []*model.LockedProvider {
	result := make([]*model.LockedProvider, 0, len(providers))
	for _, provider := range providers {
		if converted := toInternalLockedProvider(provider); converted != nil {
			result = append(result, converted)
		}
	}
	return result
}

func toInternalLockedProvider(provider *LockedProvider) *model.LockedProvider {
	if provider == nil {
		return nil
	}

	return &model.LockedProvider{
		Source:      provider.Source,
		Version:     provider.Version,
		Constraints: provider.Constraints,
	}
}

func toInternalRemoteStates(remoteStates []*RemoteStateRef) []*model.RemoteStateRef {
	result := make([]*model.RemoteStateRef, 0, len(remoteStates))
	for _, remoteState := range remoteStates {
		if converted := toInternalRemoteState(remoteState); converted != nil {
			result = append(result, converted)
		}
	}
	return result
}

func toInternalRemoteState(remoteState *RemoteStateRef) *model.RemoteStateRef {
	if remoteState == nil {
		return nil
	}

	return &model.RemoteStateRef{
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
		if converted := toInternalModuleCall(moduleCall); converted != nil {
			result = append(result, converted)
		}
	}
	return result
}

func toInternalModuleCall(moduleCall *ModuleCall) *model.ModuleCall {
	if moduleCall == nil {
		return nil
	}

	return &model.ModuleCall{
		Name:         moduleCall.Name,
		Source:       moduleCall.Source,
		Version:      moduleCall.Version,
		IsLocal:      moduleCall.IsLocal,
		ResolvedPath: moduleCall.ResolvedPath,
	}
}
