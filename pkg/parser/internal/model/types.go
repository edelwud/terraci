package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type ParsedModule struct {
	Path              string
	Locals            map[string]cty.Value
	Variables         map[string]cty.Value
	Backend           *BackendConfig
	RequiredProviders []*RequiredProvider
	LockedProviders   []*LockedProvider
	RemoteStates      []*RemoteStateRef
	ModuleCalls       []*ModuleCall
	Files             map[string]*hcl.File
	Diagnostics       hcl.Diagnostics
	TopLevelBlocks    map[string][]*hcl.Block
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

type ModuleCall struct {
	Name         string
	Source       string
	Version      string
	IsLocal      bool
	ResolvedPath string
}

type BackendConfig struct {
	Type   string
	Config map[string]string
}

type RemoteStateRef struct {
	Name         string
	Backend      string
	Config       map[string]hcl.Expression
	ForEach      hcl.Expression
	WorkspaceDir string
	RawBody      hcl.Body
}

func NewParsedModule(modulePath string) *ParsedModule {
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
		TopLevelBlocks:    make(map[string][]*hcl.Block),
	}
}

func (pm *ParsedModule) AddDiags(diags hcl.Diagnostics) {
	pm.Diagnostics = append(pm.Diagnostics, diags...)
}
