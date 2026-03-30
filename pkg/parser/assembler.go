package parser

import (
	"maps"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type moduleAssembler struct {
	parsed *ParsedModule
}

func newModuleAssembler(modulePath string, index *moduleIndex) *moduleAssembler {
	return &moduleAssembler{
		parsed: &ParsedModule{
			Path:              modulePath,
			Locals:            make(map[string]cty.Value),
			Variables:         make(map[string]cty.Value),
			RequiredProviders: make([]*RequiredProvider, 0),
			LockedProviders:   make([]*LockedProvider, 0),
			RemoteStates:      make([]*RemoteStateRef, 0),
			ModuleCalls:       make([]*ModuleCall, 0),
			Files:             cloneFiles(index.files()),
			Diagnostics:       append(hcl.Diagnostics(nil), index.diagnostics()...),
			topLevelBlocks:    cloneBlockIndex(index.topLevelBlockIndex()),
		},
	}
}

func (a *moduleAssembler) Result() *ParsedModule {
	return a.parsed
}

func cloneFiles(files map[string]*hcl.File) map[string]*hcl.File {
	cloned := make(map[string]*hcl.File, len(files))
	maps.Copy(cloned, files)
	return cloned
}

func cloneBlockIndex(index map[string][]*hcl.Block) map[string][]*hcl.Block {
	cloned := make(map[string][]*hcl.Block, len(index))
	for blockType, blocks := range index {
		cloned[blockType] = append([]*hcl.Block(nil), blocks...)
	}
	return cloned
}
