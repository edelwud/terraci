package extract

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
	"github.com/edelwud/terraci/pkg/parser/internal/source"
	"github.com/edelwud/terraci/pkg/parser/model"
)

const lockFileName = ".terraform.lock.hcl"

type Sink interface {
	Path() string
	Locals() map[string]cty.Value
	Variables() map[string]cty.Value
	AddDiags(hcl.Diagnostics)
	SetLocal(name string, value cty.Value)
	SetVariable(name string, value cty.Value)
	SetBackend(BackendConfig)
	AppendRequiredProvider(RequiredProvider)
	AppendLockedProvider(LockedProvider)
	AppendRemoteState(RemoteStateRef)
	AppendModuleCall(ModuleCall)
}

type Source interface {
	LocalsBlocks() []*hcl.Block
	VariableBlockViews() []source.VariableBlockView
	TerraformBlockViews() []source.TerraformBlockView
	RemoteStateBlockViews() []source.RemoteStateBlockView
	ModuleBlockViews() []source.ModuleBlockView
	ParseHCLFile(path string) (*hcl.File, hcl.Diagnostics, error)
}

type Context struct {
	Source      Source
	EvalBuilder evalctx.Builder
	Sink        Sink
}

type BackendConfig = model.BackendConfig

type RequiredProvider = model.RequiredProvider

type LockedProvider = model.LockedProvider

type RemoteStateRef = model.RemoteStateRef

type ModuleCall = model.ModuleCall
