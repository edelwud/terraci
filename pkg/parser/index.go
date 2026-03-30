package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

type moduleIndex struct {
	path           string
	hclParser      *hclparse.Parser
	files          map[string]*hcl.File
	topLevelBlocks map[string][]*hcl.Block
	diagnostics    hcl.Diagnostics
}

func newModuleIndex(path string, hclParser *hclparse.Parser) *moduleIndex {
	return &moduleIndex{
		path:           path,
		hclParser:      hclParser,
		files:          make(map[string]*hcl.File),
		topLevelBlocks: make(map[string][]*hcl.Block),
	}
}

func (i *moduleIndex) addDiagnostics(diags hcl.Diagnostics) {
	i.diagnostics = append(i.diagnostics, diags...)
}

func (i *moduleIndex) addFile(path string, file *hcl.File) {
	i.files[path] = file
	i.collectTopLevelBlocks(file)
}

func (i *moduleIndex) parseHCLFile(path string) (*hcl.File, error) {
	file, diags, err := parseHCLFile(i.hclParser, path)
	i.addDiagnostics(diags)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (i *moduleIndex) blocks(blockType string) []*hcl.Block {
	return i.topLevelBlocks[blockType]
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
