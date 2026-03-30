package extract

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
	"github.com/edelwud/terraci/pkg/parser/internal/source"
)

const lockFileName = ".terraform.lock.hcl"

type Sink interface {
	Path() string
	Locals() map[string]cty.Value
	Variables() map[string]cty.Value
	AddDiags(hcl.Diagnostics)
	SetLocal(name string, value cty.Value)
	SetVariable(name string, value cty.Value)
	SetBackend(Backend)
	AppendRequiredProvider(RequiredProvider)
	AppendLockedProvider(LockedProvider)
	AppendRemoteState(RemoteState)
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

type Backend struct {
	Type   string
	Config map[string]string
}

type RequiredProvider struct {
	Name              string
	Source            string
	VersionConstraint string
}

type LockedProvider struct {
	Source      string
	Version     string
	Constraints string
}

type RemoteState struct {
	Name    string
	Backend string
	Config  map[string]hcl.Expression
	ForEach hcl.Expression
	RawBody hcl.Body
}

type ModuleCall struct {
	Name         string
	Source       string
	Version      string
	IsLocal      bool
	ResolvedPath string
}
