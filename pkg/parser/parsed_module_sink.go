package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/extract"
	moduleparse "github.com/edelwud/terraci/pkg/parser/internal/moduleparse"
)

type parsedModuleSink struct {
	parsed *ParsedModule
}

func newParsedModule(modulePath string) *ParsedModule {
	return &ParsedModule{
		Path:              modulePath,
		Locals:            make(map[string]cty.Value),
		Variables:         make(map[string]cty.Value),
		RequiredProviders: make([]*RequiredProvider, 0),
		LockedProviders:   make([]*LockedProvider, 0),
		RemoteStates:      make([]*RemoteStateRef, 0),
		ModuleCalls:       make([]*ModuleCall, 0),
		Files:             make(map[string]*hcl.File),
		Diagnostics:       make(hcl.Diagnostics, 0),
		topLevelBlocks:    make(map[string][]*hcl.Block),
	}
}

func newParsedModuleSink(parsed *ParsedModule) *parsedModuleSink {
	return &parsedModuleSink{parsed: parsed}
}

func (s *parsedModuleSink) AddDiags(diags hcl.Diagnostics) {
	s.parsed.addDiags(diags)
}

func (s *parsedModuleSink) Path() string {
	return s.parsed.Path
}

func (s *parsedModuleSink) Locals() map[string]cty.Value {
	return s.parsed.Locals
}

func (s *parsedModuleSink) Variables() map[string]cty.Value {
	return s.parsed.Variables
}

func (s *parsedModuleSink) SetLocal(name string, value cty.Value) {
	s.parsed.Locals[name] = value
}

func (s *parsedModuleSink) SetVariable(name string, value cty.Value) {
	s.parsed.Variables[name] = value
}

func (s *parsedModuleSink) SetBackend(backend extract.Backend) {
	s.parsed.Backend = &BackendConfig{
		Type:   backend.Type,
		Config: backend.Config,
	}
}

func (s *parsedModuleSink) AppendRequiredProvider(provider extract.RequiredProvider) {
	s.parsed.RequiredProviders = append(s.parsed.RequiredProviders, &RequiredProvider{
		Name:              provider.Name,
		Source:            provider.Source,
		VersionConstraint: provider.VersionConstraint,
	})
}

func (s *parsedModuleSink) AppendLockedProvider(provider extract.LockedProvider) {
	s.parsed.LockedProviders = append(s.parsed.LockedProviders, &LockedProvider{
		Source:      provider.Source,
		Version:     provider.Version,
		Constraints: provider.Constraints,
	})
}

func (s *parsedModuleSink) AppendRemoteState(ref extract.RemoteState) {
	s.parsed.RemoteStates = append(s.parsed.RemoteStates, &RemoteStateRef{
		Name:    ref.Name,
		Backend: ref.Backend,
		Config:  ref.Config,
		ForEach: ref.ForEach,
		RawBody: ref.RawBody,
	})
}

func (s *parsedModuleSink) AppendModuleCall(call extract.ModuleCall) {
	s.parsed.ModuleCalls = append(s.parsed.ModuleCalls, &ModuleCall{
		Name:         call.Name,
		Source:       call.Source,
		Version:      call.Version,
		IsLocal:      call.IsLocal,
		ResolvedPath: call.ResolvedPath,
	})
}

func applyParseResult(parsed *ParsedModule, result *moduleparse.Result) {
	if result == nil {
		return
	}

	parsed.Files = result.Files
	parsed.Diagnostics = append(parsed.Diagnostics, result.Diagnostics...)
	parsed.topLevelBlocks = result.TopLevelBlocks
}
