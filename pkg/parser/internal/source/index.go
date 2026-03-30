package source

import (
	"maps"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

type Index struct {
	path           string
	hclParser      *hclparse.Parser
	files          map[string]*hcl.File
	topLevelBlocks map[string][]*hcl.Block
	diagnostics    hcl.Diagnostics
}

func NewIndex(path string, hclParser *hclparse.Parser) *Index {
	return &Index{
		path:           path,
		hclParser:      hclParser,
		files:          make(map[string]*hcl.File),
		topLevelBlocks: make(map[string][]*hcl.Block),
	}
}

func (i *Index) AddDiagnostics(diags hcl.Diagnostics) {
	i.diagnostics = append(i.diagnostics, diags...)
}

func (i *Index) AddFile(path string, file *hcl.File) {
	i.files[path] = file
	i.collectTopLevelBlocks(file)
}

func (i *Index) ParseHCLFile(path string) (*hcl.File, error) {
	file, diags, err := parseHCLFile(i.hclParser, path)
	i.AddDiagnostics(diags)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (i *Index) Files() map[string]*hcl.File {
	cloned := make(map[string]*hcl.File, len(i.files))
	maps.Copy(cloned, i.files)
	return cloned
}

func (i *Index) Diagnostics() hcl.Diagnostics {
	return append(hcl.Diagnostics(nil), i.diagnostics...)
}

func (i *Index) TopLevelBlockIndex() map[string][]*hcl.Block {
	cloned := make(map[string][]*hcl.Block, len(i.topLevelBlocks))
	for blockType, blocks := range i.topLevelBlocks {
		cloned[blockType] = append([]*hcl.Block(nil), blocks...)
	}
	return cloned
}

func (i *Index) Blocks(blockType string) []*hcl.Block {
	return i.topLevelBlocks[blockType]
}

func (i *Index) LocalsBlocks() []*hcl.Block {
	return i.Blocks("locals")
}

func (i *Index) VariableBlocks() []*hcl.Block {
	return i.Blocks("variable")
}

func (i *Index) TerraformBlocks() []*hcl.Block {
	return i.Blocks("terraform")
}

func (i *Index) DataBlocks() []*hcl.Block {
	return i.Blocks("data")
}

func (i *Index) ModuleBlocks() []*hcl.Block {
	return i.Blocks("module")
}
