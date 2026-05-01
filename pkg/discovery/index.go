package discovery

// ModuleIndex provides O(1) lookup of modules by ID and by path. The index is
// built once over a fixed module slice; All() returns the source slice in
// registration order.
type ModuleIndex struct {
	modules []*Module
	byID    map[string]*Module
	byPath  map[string]*Module
}

// NewModuleIndex creates an index from a list of modules.
func NewModuleIndex(modules []*Module) *ModuleIndex {
	idx := &ModuleIndex{
		modules: modules,
		byID:    make(map[string]*Module, len(modules)),
		byPath:  make(map[string]*Module, len(modules)),
	}

	for _, m := range modules {
		idx.byID[m.ID()] = m
		idx.byPath[m.Path] = m
		idx.byPath[m.RelativePath] = m
	}

	return idx
}

// All returns all modules.
func (idx *ModuleIndex) All() []*Module { return idx.modules }

// ByID returns a module by its ID.
func (idx *ModuleIndex) ByID(id string) *Module { return idx.byID[id] }

// ByPath returns a module by its path.
func (idx *ModuleIndex) ByPath(path string) *Module { return idx.byPath[path] }
