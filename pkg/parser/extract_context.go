package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/extract"
)

type extractContext struct {
	index       *moduleIndex
	parsed      *ParsedModule
	evalBuilder evalContextBuilder
}

func newExtractContext(index *moduleIndex, parsed *ParsedModule, evalBuilder evalContextBuilder) *extractContext {
	return &extractContext{
		index:       index,
		parsed:      parsed,
		evalBuilder: evalBuilder,
	}
}

func (c *extractContext) AddDiags(diags hcl.Diagnostics) {
	c.parsed.addDiags(diags)
}

func (c *extractContext) Path() string {
	return c.parsed.Path
}

func (c *extractContext) Locals() map[string]cty.Value {
	return c.parsed.Locals
}

func (c *extractContext) Variables() map[string]cty.Value {
	return c.parsed.Variables
}

func (c *extractContext) SetLocal(name string, value cty.Value) {
	c.parsed.Locals[name] = value
}

func (c *extractContext) SetVariable(name string, value cty.Value) {
	c.parsed.Variables[name] = value
}

func (c *extractContext) SetBackend(backend extract.Backend) {
	c.parsed.Backend = &BackendConfig{
		Type:   backend.Type,
		Config: backend.Config,
	}
}

func (c *extractContext) AppendRequiredProvider(provider extract.RequiredProvider) {
	c.parsed.RequiredProviders = append(c.parsed.RequiredProviders, &RequiredProvider{
		Name:              provider.Name,
		Source:            provider.Source,
		VersionConstraint: provider.VersionConstraint,
	})
}

func (c *extractContext) AppendLockedProvider(provider extract.LockedProvider) {
	c.parsed.LockedProviders = append(c.parsed.LockedProviders, &LockedProvider{
		Source:      provider.Source,
		Version:     provider.Version,
		Constraints: provider.Constraints,
	})
}

func (c *extractContext) AppendRemoteState(ref extract.RemoteState) {
	c.parsed.RemoteStates = append(c.parsed.RemoteStates, &RemoteStateRef{
		Name:    ref.Name,
		Backend: ref.Backend,
		Config:  ref.Config,
		ForEach: ref.ForEach,
		RawBody: ref.RawBody,
	})
}

func (c *extractContext) AppendModuleCall(call extract.ModuleCall) {
	c.parsed.ModuleCalls = append(c.parsed.ModuleCalls, &ModuleCall{
		Name:         call.Name,
		Source:       call.Source,
		Version:      call.Version,
		IsLocal:      call.IsLocal,
		ResolvedPath: call.ResolvedPath,
	})
}

func (c *extractContext) extractionContext() *extract.Context {
	return &extract.Context{
		Index:       c.index.inner,
		EvalBuilder: c.evalBuilder.inner,
		Sink:        c,
	}
}
