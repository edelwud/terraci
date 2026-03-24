package discovery

import "path/filepath"

// ModuleIndex provides fast lookup of modules by various keys.
type ModuleIndex struct {
	modules    []*Module
	byID       map[string]*Module
	byPath     map[string]*Module
	byBaseName map[string][]*Module
}

// NewModuleIndex creates an index from a list of modules.
func NewModuleIndex(modules []*Module) *ModuleIndex {
	idx := &ModuleIndex{
		modules:    modules,
		byID:       make(map[string]*Module, len(modules)),
		byPath:     make(map[string]*Module, len(modules)),
		byBaseName: make(map[string][]*Module),
	}

	for _, m := range modules {
		idx.byID[m.ID()] = m
		idx.byPath[m.Path] = m
		idx.byPath[m.RelativePath] = m

		baseName := m.LeafValue()
		idx.byBaseName[baseName] = append(idx.byBaseName[baseName], m)

		if m.IsSubmodule() {
			idx.byBaseName[m.Name()] = append(idx.byBaseName[m.Name()], m)
		}
	}

	return idx
}

// All returns all modules.
func (idx *ModuleIndex) All() []*Module { return idx.modules }

// ByID returns a module by its ID.
func (idx *ModuleIndex) ByID(id string) *Module { return idx.byID[id] }

// ByPath returns a module by its path.
func (idx *ModuleIndex) ByPath(path string) *Module { return idx.byPath[path] }

// ByName returns all modules with the given base name.
func (idx *ModuleIndex) ByName(name string) []*Module { return idx.byBaseName[name] }

// Filter returns modules matching the given predicate.
func (idx *ModuleIndex) Filter(fn func(*Module) bool) []*Module {
	var result []*Module
	for _, m := range idx.modules {
		if fn(m) {
			result = append(result, m)
		}
	}
	return result
}

// BaseModules returns only non-submodule modules.
func (idx *ModuleIndex) BaseModules() []*Module {
	return idx.Filter(func(m *Module) bool { return !m.IsSubmodule() })
}

// Submodules returns only submodules.
func (idx *ModuleIndex) Submodules() []*Module {
	return idx.Filter(func(m *Module) bool { return m.IsSubmodule() })
}

// FindInContext tries to find a module by name within the same context.
func (idx *ModuleIndex) FindInContext(name string, context *Module) *Module {
	contextPrefix := context.ContextPrefix()

	// Try exact match in same context
	if m := idx.byID[filepath.Join(contextPrefix, name)]; m != nil {
		return m
	}

	// Try as submodule of current leaf
	if leaf := context.LeafValue(); leaf != "" {
		if m := idx.byID[filepath.Join(contextPrefix, leaf, name)]; m != nil {
			return m
		}
	}

	// Try by name in same context
	for _, m := range idx.byBaseName[name] {
		if m.ContextPrefix() == contextPrefix {
			return m
		}
	}

	return nil
}
