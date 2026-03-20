// Package discovery provides functionality for discovering Terraform modules
// in a directory structure following a configurable pattern like: service/environment/region/module[/submodule]
package discovery

import (
	"path/filepath"
)

// Module represents a discovered Terraform module with its path components
type Module struct {
	components map[string]string
	segments   []string
	// Full path to the module directory
	Path string
	// Relative path from the root directory
	RelativePath string
	// Parent module reference (for submodules)
	Parent *Module
	// Children submodules
	Children []*Module
}

// NewModule creates a Module from ordered segment names and values.
func NewModule(segments, values []string, path, relPath string) *Module {
	components := make(map[string]string, len(segments))
	for i := range segments {
		if i < len(values) {
			components[segments[i]] = values[i]
		}
	}
	return &Module{
		components:   components,
		segments:     segments,
		Path:         path,
		RelativePath: relPath,
	}
}

// TestModule creates a Module with the default pattern for testing.
func TestModule(service, env, region, module string) *Module {
	segments := []string{"service", "environment", "region", "module"}
	values := []string{service, env, region, module}
	relPath := filepath.Join(service, env, region, module)
	return NewModule(segments, values, relPath, relPath)
}

// Get returns the value of a named component.
func (m *Module) Get(name string) string { return m.components[name] }

// LeafValue returns the value of the last pattern segment (the "module" equivalent).
func (m *Module) LeafValue() string {
	if len(m.segments) == 0 {
		return ""
	}
	return m.components[m.segments[len(m.segments)-1]]
}

// Segments returns the ordered segment names.
func (m *Module) Segments() []string { return m.segments }

// Components returns a copy of the component map.
func (m *Module) Components() map[string]string {
	cp := make(map[string]string, len(m.components))
	for k, v := range m.components {
		cp[k] = v
	}
	return cp
}

// SetComponent sets a component value (used internally for submodule assignment).
func (m *Module) SetComponent(name, value string) {
	m.components[name] = value
}

// Name returns the full module name including submodule if present
func (m *Module) Name() string {
	// Last segment value is the "module" name
	leaf := ""
	if len(m.segments) > 0 {
		leaf = m.components[m.segments[len(m.segments)-1]]
	}
	if m.Get("submodule") != "" {
		return leaf + "/" + m.Get("submodule")
	}
	return leaf
}

// ID returns a unique identifier for the module (its relative path).
func (m *Module) ID() string {
	return m.RelativePath
}

// String returns the module ID
func (m *Module) String() string {
	return m.ID()
}

// IsSubmodule returns true if this module is a submodule
func (m *Module) IsSubmodule() bool {
	return m.Get("submodule") != ""
}

// ContextPrefix returns the path prefix for context-relative lookups
// (all segments except the last, joined with /)
func (m *Module) ContextPrefix() string {
	if len(m.segments) <= 1 {
		return ""
	}
	parts := make([]string, len(m.segments)-1)
	for i, seg := range m.segments[:len(m.segments)-1] {
		parts[i] = m.components[seg]
	}
	return filepath.Join(parts...)
}

// Scanner discovers Terraform modules in a directory tree
type Scanner struct {
	// RootDir is the root directory to scan
	RootDir string
	// MinDepth defines minimum directory depth (default: 4 for service/env/region/module)
	MinDepth int
	// MaxDepth defines maximum directory depth (default: 5 for service/env/region/module/submodule)
	MaxDepth int
	// Segments are the ordered pattern segment names (default: service, environment, region, module)
	Segments []string
}

// NewScanner creates a new Scanner with the given root directory
func NewScanner(rootDir string) *Scanner {
	return &Scanner{
		RootDir:  rootDir,
		MinDepth: 4,
		MaxDepth: 5,
		Segments: []string{"service", "environment", "region", "module"},
	}
}

// Scan walks the directory tree and returns all discovered Terraform modules
func (s *Scanner) Scan() ([]*Module, error) {
	absRoot, err := filepath.Abs(s.RootDir)
	if err != nil {
		return nil, err
	}

	collector := &moduleCollector{
		absRoot:  absRoot,
		minDepth: s.MinDepth,
		maxDepth: s.MaxDepth,
		segments: s.Segments,
		byID:     make(map[string]*Module),
	}

	if err := filepath.Walk(absRoot, collector.visit); err != nil {
		return nil, err
	}

	return collector.modules, nil
}

// ModuleIndex provides fast lookup of modules by various keys
type ModuleIndex struct {
	modules    []*Module
	byID       map[string]*Module
	byPath     map[string]*Module
	byBaseName map[string][]*Module // module name -> all modules with that name
}

// NewModuleIndex creates an index from a list of modules
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

		// Index by base module name (last segment value)
		baseName := m.LeafValue()
		idx.byBaseName[baseName] = append(idx.byBaseName[baseName], m)

		// Also index by full name (module/submodule) if it's a submodule
		if m.IsSubmodule() {
			idx.byBaseName[m.Name()] = append(idx.byBaseName[m.Name()], m)
		}
	}

	return idx
}

// All returns all modules
func (idx *ModuleIndex) All() []*Module {
	return idx.modules
}

// ByID returns a module by its ID
func (idx *ModuleIndex) ByID(id string) *Module {
	return idx.byID[id]
}

// ByPath returns a module by its path
func (idx *ModuleIndex) ByPath(path string) *Module {
	return idx.byPath[path]
}

// ByName returns all modules with the given base name
func (idx *ModuleIndex) ByName(name string) []*Module {
	return idx.byBaseName[name]
}

// Filter returns modules matching the given filter function
func (idx *ModuleIndex) Filter(fn func(*Module) bool) []*Module {
	var result []*Module
	for _, m := range idx.modules {
		if fn(m) {
			result = append(result, m)
		}
	}
	return result
}

// BaseModules returns only non-submodule modules
func (idx *ModuleIndex) BaseModules() []*Module {
	return idx.Filter(func(m *Module) bool {
		return !m.IsSubmodule()
	})
}

// Submodules returns only submodules
func (idx *ModuleIndex) Submodules() []*Module {
	return idx.Filter(func(m *Module) bool {
		return m.IsSubmodule()
	})
}

// FindInContext tries to find a module by name within the same context
func (idx *ModuleIndex) FindInContext(name string, context *Module) *Module {
	// Try exact match in same context
	contextPrefix := context.ContextPrefix()
	candidates := []string{
		filepath.Join(contextPrefix, name),
	}
	// Also try as submodule
	if leafVal := context.LeafValue(); leafVal != "" {
		candidates = append(candidates, filepath.Join(contextPrefix, leafVal, name))
	}
	for _, id := range candidates {
		if m := idx.byID[id]; m != nil {
			return m
		}
	}
	// Try by name in same context
	modules := idx.byBaseName[name]
	for _, m := range modules {
		if m.ContextPrefix() == contextPrefix {
			return m
		}
	}
	return nil
}
