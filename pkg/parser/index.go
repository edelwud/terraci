package parser

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/pkg/parser/internal/source"
)

type moduleIndex struct {
	inner *source.Index
}

func newModuleIndex(inner *source.Index) *moduleIndex {
	return &moduleIndex{inner: inner}
}

func (i *moduleIndex) blocks(blockType string) []*hcl.Block {
	return i.inner.Blocks(blockType)
}

func (i *moduleIndex) localsBlocks() []*hcl.Block {
	return i.blocks("locals")
}

func (i *moduleIndex) variableBlocks() []*hcl.Block {
	return i.blocks("variable")
}

func (i *moduleIndex) terraformBlocks() []*hcl.Block {
	return i.blocks("terraform")
}

func (i *moduleIndex) dataBlocks() []*hcl.Block {
	return i.blocks("data")
}

func (i *moduleIndex) moduleBlocks() []*hcl.Block {
	return i.blocks("module")
}

func (i *moduleIndex) files() map[string]*hcl.File {
	return i.inner.Files()
}

func (i *moduleIndex) diagnostics() hcl.Diagnostics {
	return i.inner.Diagnostics()
}

func (i *moduleIndex) topLevelBlockIndex() map[string][]*hcl.Block {
	return i.inner.TopLevelBlockIndex()
}
