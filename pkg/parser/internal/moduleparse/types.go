package moduleparse

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
	"github.com/edelwud/terraci/pkg/parser/internal/extract"
	"github.com/edelwud/terraci/pkg/parser/internal/source"
	"github.com/edelwud/terraci/pkg/parser/model"
)

type loadedSource interface {
	extract.Source
	SharedFiles() map[string]*hcl.File
	SharedDiagnostics() hcl.Diagnostics
	SharedTopLevelBlockIndex() map[string][]*hcl.Block
}

type sourceLoader interface {
	Load(context.Context, string) (loadedSource, error)
}

type defaultSourceLoader struct{}

func (defaultSourceLoader) Load(ctx context.Context, modulePath string) (loadedSource, error) {
	return source.NewLoader().Load(ctx, modulePath)
}

type runner struct {
	modulePath string
	loader     sourceLoader
	source     loadedSource
	parsed     *model.ParsedModule
	extractCtx *extract.Context
}

func newRunner(modulePath string, segments []string) *runner {
	return newRunnerWithLoader(modulePath, segments, defaultSourceLoader{})
}

func newRunnerWithLoader(modulePath string, segments []string, loader sourceLoader) *runner {
	parsed := model.NewParsedModule(modulePath)
	sink := &parsedModuleSink{parsed: parsed}

	return &runner{
		modulePath: modulePath,
		loader:     loader,
		parsed:     parsed,
		extractCtx: &extract.Context{
			Source:      nil,
			EvalBuilder: evalctx.NewBuilder(segments),
			Sink:        sink,
		},
	}
}

type parsedModuleSink struct {
	parsed *model.ParsedModule
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

func (s *parsedModuleSink) SetBackend(backend extract.BackendConfig) {
	s.parsed.Backend = &backend
}

func (s *parsedModuleSink) AppendRequiredProvider(provider extract.RequiredProvider) {
	s.parsed.RequiredProviders = append(s.parsed.RequiredProviders, &provider)
}

func (s *parsedModuleSink) AppendLockedProvider(provider extract.LockedProvider) {
	s.parsed.LockedProviders = append(s.parsed.LockedProviders, &provider)
}

func (s *parsedModuleSink) AppendRemoteState(ref extract.RemoteStateRef) {
	s.parsed.RemoteStates = append(s.parsed.RemoteStates, &ref)
}

func (s *parsedModuleSink) AppendModuleCall(call extract.ModuleCall) {
	s.parsed.ModuleCalls = append(s.parsed.ModuleCalls, &call)
}

func (r *runner) Run(ctx context.Context) (*model.ParsedModule, error) {
	if err := r.load(ctx); err != nil {
		return nil, err
	}

	r.extract()
	r.finalize()

	return r.parsed, nil
}

func (r *runner) load(ctx context.Context) error {
	src, err := r.loader.Load(ctx, r.modulePath)
	if err != nil {
		return err
	}

	r.source = src
	r.extractCtx.Source = src
	return nil
}

func (r *runner) extract() {
	extract.RunDefault(r.extractCtx)
}

func (r *runner) finalize() {
	r.parsed.Files = r.source.SharedFiles()
	r.parsed.AddDiags(r.source.SharedDiagnostics())
	r.parsed.AdoptTopLevelBlocks(r.source.SharedTopLevelBlockIndex())
}
