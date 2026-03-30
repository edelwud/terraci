package moduleparse

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
	"github.com/edelwud/terraci/pkg/parser/internal/extract"
	"github.com/edelwud/terraci/pkg/parser/internal/model"
	"github.com/edelwud/terraci/pkg/parser/internal/source"
)

type parsedModuleSink struct {
	parsed *model.ParsedModule
}

func Run(ctx context.Context, modulePath string, segments []string) (*model.ParsedModule, error) {
	index, err := source.NewLoader().Load(ctx, modulePath)
	if err != nil {
		return nil, err
	}

	parsed := model.NewParsedModule(modulePath)
	sink := &parsedModuleSink{parsed: parsed}
	extract.RunDefault(&extract.Context{
		Index:       index,
		EvalBuilder: evalctx.NewBuilder(segments),
		Sink:        sink,
	})

	parsed.Files = index.Files()
	parsed.AddDiags(index.Diagnostics())
	parsed.TopLevelBlocks = index.TopLevelBlockIndex()

	return parsed, nil
}

func (s *parsedModuleSink) AddDiags(diags hcl.Diagnostics) {
	s.parsed.AddDiags(diags)
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
	s.parsed.Backend = &model.BackendConfig{
		Type:   backend.Type,
		Config: backend.Config,
	}
}

func (s *parsedModuleSink) AppendRequiredProvider(provider extract.RequiredProvider) {
	s.parsed.RequiredProviders = append(s.parsed.RequiredProviders, &model.RequiredProvider{
		Name:              provider.Name,
		Source:            provider.Source,
		VersionConstraint: provider.VersionConstraint,
	})
}

func (s *parsedModuleSink) AppendLockedProvider(provider extract.LockedProvider) {
	s.parsed.LockedProviders = append(s.parsed.LockedProviders, &model.LockedProvider{
		Source:      provider.Source,
		Version:     provider.Version,
		Constraints: provider.Constraints,
	})
}

func (s *parsedModuleSink) AppendRemoteState(ref extract.RemoteState) {
	s.parsed.RemoteStates = append(s.parsed.RemoteStates, &model.RemoteStateRef{
		Name:    ref.Name,
		Backend: ref.Backend,
		Config:  ref.Config,
		ForEach: ref.ForEach,
		RawBody: ref.RawBody,
	})
}

func (s *parsedModuleSink) AppendModuleCall(call extract.ModuleCall) {
	s.parsed.ModuleCalls = append(s.parsed.ModuleCalls, &model.ModuleCall{
		Name:         call.Name,
		Source:       call.Source,
		Version:      call.Version,
		IsLocal:      call.IsLocal,
		ResolvedPath: call.ResolvedPath,
	})
}
