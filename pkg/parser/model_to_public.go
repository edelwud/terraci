package parser

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/pkg/parser/internal/model"
)

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

func fromInternalBackend(backend *model.BackendConfig) *BackendConfig {
	if backend == nil {
		return nil
	}

	return &BackendConfig{
		Type:   backend.Type,
		Config: backend.Config,
	}
}

func fromInternalRequiredProviders(providers []*model.RequiredProvider) []*RequiredProvider {
	result := make([]*RequiredProvider, 0, len(providers))
	for _, provider := range providers {
		if converted := fromInternalRequiredProvider(provider); converted != nil {
			result = append(result, converted)
		}
	}
	return result
}

func fromInternalRequiredProvider(provider *model.RequiredProvider) *RequiredProvider {
	if provider == nil {
		return nil
	}

	return &RequiredProvider{
		Name:              provider.Name,
		Source:            provider.Source,
		VersionConstraint: provider.VersionConstraint,
	}
}

func fromInternalLockedProviders(providers []*model.LockedProvider) []*LockedProvider {
	result := make([]*LockedProvider, 0, len(providers))
	for _, provider := range providers {
		if converted := fromInternalLockedProvider(provider); converted != nil {
			result = append(result, converted)
		}
	}
	return result
}

func fromInternalLockedProvider(provider *model.LockedProvider) *LockedProvider {
	if provider == nil {
		return nil
	}

	return &LockedProvider{
		Source:      provider.Source,
		Version:     provider.Version,
		Constraints: provider.Constraints,
	}
}

func fromInternalRemoteStates(remoteStates []*model.RemoteStateRef) []*RemoteStateRef {
	result := make([]*RemoteStateRef, 0, len(remoteStates))
	for _, remoteState := range remoteStates {
		if converted := fromInternalRemoteState(remoteState); converted != nil {
			result = append(result, converted)
		}
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

func fromInternalModuleCalls(moduleCalls []*model.ModuleCall) []*ModuleCall {
	result := make([]*ModuleCall, 0, len(moduleCalls))
	for _, moduleCall := range moduleCalls {
		if converted := fromInternalModuleCall(moduleCall); converted != nil {
			result = append(result, converted)
		}
	}
	return result
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
