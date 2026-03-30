package source

import (
	"maps"

	"github.com/hashicorp/hcl/v2"
)

type Snapshot struct {
	path           string
	files          map[string]*hcl.File
	topLevelBlocks map[string][]*hcl.Block
	variableViews  []VariableBlockView
	terraformViews []TerraformBlockView
	moduleViews    []ModuleBlockView
	remoteViews    []RemoteStateBlockView
	lockFile       *hcl.File
	lockFileDiags  hcl.Diagnostics
	diagnostics    hcl.Diagnostics
}

type indexBuilder struct {
	snapshot *Snapshot
}

func newSnapshot(path string, fileCap int) *Snapshot {
	if fileCap < 0 {
		fileCap = 0
	}

	return &Snapshot{
		path:           path,
		files:          make(map[string]*hcl.File, fileCap),
		topLevelBlocks: make(map[string][]*hcl.Block, 5),
		variableViews:  make([]VariableBlockView, 0, fileCap),
		terraformViews: make([]TerraformBlockView, 0, fileCap),
		moduleViews:    make([]ModuleBlockView, 0, fileCap),
		remoteViews:    make([]RemoteStateBlockView, 0, fileCap),
	}
}

func newIndexBuilder(path string, fileCap int) *indexBuilder {
	return &indexBuilder{
		snapshot: newSnapshot(path, fileCap),
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
	file, diags, err := parseHCLFile(path)
	b.AddDiagnostics(diags)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (b *indexBuilder) Snapshot() *Snapshot {
	return b.snapshot
}

func (b *indexBuilder) SetLockFile(file *hcl.File, diags hcl.Diagnostics) {
	b.snapshot.lockFile = file
	b.snapshot.lockFileDiags = diags
}

func (s *Snapshot) SharedFiles() map[string]*hcl.File {
	return s.files
}

func (s *Snapshot) Files() map[string]*hcl.File {
	cloned := make(map[string]*hcl.File, len(s.files))
	maps.Copy(cloned, s.files)
	return cloned
}

func (s *Snapshot) ParseHCLFile(path string) (*hcl.File, hcl.Diagnostics, error) {
	file, diags, err := parseHCLFile(path)
	if err != nil {
		return nil, diags, err
	}
	return file, diags, nil
}

func (s *Snapshot) LockFile() (*hcl.File, hcl.Diagnostics) {
	return s.lockFile, append(hcl.Diagnostics(nil), s.lockFileDiags...)
}

func (s *Snapshot) Diagnostics() hcl.Diagnostics {
	return append(hcl.Diagnostics(nil), s.diagnostics...)
}

func (s *Snapshot) SharedDiagnostics() hcl.Diagnostics {
	return s.diagnostics
}

func (s *Snapshot) SharedTopLevelBlockIndex() map[string][]*hcl.Block {
	return s.topLevelBlocks
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
