package source

import (
	"maps"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

type Snapshot struct {
	path           string
	hclParser      *hclparse.Parser
	files          map[string]*hcl.File
	topLevelBlocks map[string][]*hcl.Block
	diagnostics    hcl.Diagnostics
}

type indexBuilder struct {
	hclParser *hclparse.Parser
	snapshot  *Snapshot
}

func newSnapshot(path string) *Snapshot {
	return &Snapshot{
		path:           path,
		files:          make(map[string]*hcl.File),
		topLevelBlocks: make(map[string][]*hcl.Block),
	}
}

func newIndexBuilder(path string, hclParser *hclparse.Parser) *indexBuilder {
	snapshot := newSnapshot(path)
	snapshot.hclParser = hclParser

	return &indexBuilder{
		hclParser: hclParser,
		snapshot:  snapshot,
	}
}

func (b *indexBuilder) AddDiagnostics(diags hcl.Diagnostics) {
	b.snapshot.diagnostics = append(b.snapshot.diagnostics, diags...)
}

func (b *indexBuilder) AddFile(path string, file *hcl.File) {
	b.snapshot.files[path] = file
	b.collectTopLevelBlocks(file)
}

func (b *indexBuilder) ParseHCLFile(path string) (*hcl.File, error) {
	file, diags, err := parseHCLFile(b.hclParser, path)
	b.AddDiagnostics(diags)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (b *indexBuilder) Snapshot() *Snapshot {
	return b.snapshot
}

func (s *Snapshot) Files() map[string]*hcl.File {
	cloned := make(map[string]*hcl.File, len(s.files))
	maps.Copy(cloned, s.files)
	return cloned
}

func (s *Snapshot) ParseHCLFile(path string) (*hcl.File, hcl.Diagnostics, error) {
	file, diags, err := parseHCLFile(s.hclParser, path)
	if err != nil {
		return nil, diags, err
	}
	return file, diags, nil
}

func (s *Snapshot) Diagnostics() hcl.Diagnostics {
	return append(hcl.Diagnostics(nil), s.diagnostics...)
}

func (s *Snapshot) TopLevelBlockIndex() map[string][]*hcl.Block {
	cloned := make(map[string][]*hcl.Block, len(s.topLevelBlocks))
	for blockType, blocks := range s.topLevelBlocks {
		cloned[blockType] = append([]*hcl.Block(nil), blocks...)
	}
	return cloned
}

func (s *Snapshot) Blocks(blockType string) []*hcl.Block {
	return s.topLevelBlocks[blockType]
}

func (s *Snapshot) LocalsBlocks() []*hcl.Block {
	return s.Blocks("locals")
}

func (s *Snapshot) VariableBlocks() []*hcl.Block {
	return s.Blocks("variable")
}

func (s *Snapshot) TerraformBlocks() []*hcl.Block {
	return s.Blocks("terraform")
}

func (s *Snapshot) DataBlocks() []*hcl.Block {
	return s.Blocks("data")
}

func (s *Snapshot) ModuleBlocks() []*hcl.Block {
	return s.Blocks("module")
}
