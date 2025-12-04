// Package discovery provides functionality for discovering Terraform modules
// in a directory structure following the pattern: service/environment/region/module[/submodule]
package discovery

import (
	"os"
	"path/filepath"
	"strings"
)

// Module represents a discovered Terraform module with its path components
type Module struct {
	// Service name (e.g., "cdp")
	Service string
	// Environment name (e.g., "stage", "prod")
	Environment string
	// Region name (e.g., "eu-central-1")
	Region string
	// Module name (e.g., "vpc", "eks", "ec2")
	Module string
	// Submodule name (optional, e.g., "rabbitmq" for ec2/rabbitmq)
	Submodule string
	// Full path to the module directory
	Path string
	// Relative path from the root directory
	RelativePath string
	// Parent module reference (for submodules)
	Parent *Module
	// Children submodules
	Children []*Module
}

// Name returns the full module name including submodule if present
func (m *Module) Name() string {
	if m.Submodule != "" {
		return m.Module + "/" + m.Submodule
	}
	return m.Module
}

// ID returns a unique identifier for the module
// Format: service/environment/region/module or service/environment/region/module/submodule
func (m *Module) ID() string {
	if m.Submodule != "" {
		return filepath.Join(m.Service, m.Environment, m.Region, m.Module, m.Submodule)
	}
	return filepath.Join(m.Service, m.Environment, m.Region, m.Module)
}

// String returns the module ID
func (m *Module) String() string {
	return m.ID()
}

// IsSubmodule returns true if this module is a submodule
func (m *Module) IsSubmodule() bool {
	return m.Submodule != ""
}

// BaseModule returns the module name without submodule
func (m *Module) BaseModule() string {
	return m.Module
}

// Scanner discovers Terraform modules in a directory tree
type Scanner struct {
	// RootDir is the root directory to scan
	RootDir string
	// MinDepth defines minimum directory depth (default: 4 for service/env/region/module)
	MinDepth int
	// MaxDepth defines maximum directory depth (default: 5 for service/env/region/module/submodule)
	MaxDepth int
}

// NewScanner creates a new Scanner with the given root directory
func NewScanner(rootDir string) *Scanner {
	return &Scanner{
		RootDir:  rootDir,
		MinDepth: 4,
		MaxDepth: 5,
	}
}

// Scan walks the directory tree and returns all discovered Terraform modules
func (s *Scanner) Scan() ([]*Module, error) {
	var modules []*Module
	moduleMap := make(map[string]*Module) // For linking parents

	absRoot, err := filepath.Abs(s.RootDir)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// We're looking for directories that contain .tf files
		if !info.IsDir() {
			return nil
		}

		// Check if this directory contains Terraform files
		if !containsTerraformFiles(path) {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Parse the path components
		parts := strings.Split(relPath, string(os.PathSeparator))
		depth := len(parts)

		// Check depth constraints
		if depth < s.MinDepth || depth > s.MaxDepth {
			// If depth is less than minimum, continue scanning deeper
			if depth < s.MinDepth {
				return nil
			}
			// If depth exceeds maximum, skip this subtree
			return filepath.SkipDir
		}

		module := &Module{
			Service:      parts[0],
			Environment:  parts[1],
			Region:       parts[2],
			Module:       parts[3],
			Path:         path,
			RelativePath: relPath,
		}

		// Handle submodule (depth 5)
		if depth == 5 {
			module.Submodule = parts[4]

			// Link to parent module if exists
			parentID := filepath.Join(parts[0], parts[1], parts[2], parts[3])
			if parent, ok := moduleMap[parentID]; ok {
				module.Parent = parent
				parent.Children = append(parent.Children, module)
			}
		}

		moduleMap[module.ID()] = module
		modules = append(modules, module)

		// If this is a base module (depth 4), continue scanning for submodules
		// If this is a submodule (depth 5), don't go deeper
		if depth >= s.MaxDepth {
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return modules, nil
}

// containsTerraformFiles checks if a directory contains .tf files
func containsTerraformFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") {
			return true
		}
	}

	return false
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

		// Index by base module name
		idx.byBaseName[m.Module] = append(idx.byBaseName[m.Module], m)

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

// FindInContext tries to find a module by name within the same context (service/env/region)
func (idx *ModuleIndex) FindInContext(name string, context *Module) *Module {
	// First try exact match in same context
	candidates := []string{
		// Same service/env/region + name
		filepath.Join(context.Service, context.Environment, context.Region, name),
		// Same service/env/region + module/submodule
		filepath.Join(context.Service, context.Environment, context.Region, context.Module, name),
	}

	for _, id := range candidates {
		if m := idx.byID[id]; m != nil {
			return m
		}
	}

	// Try by name in same context
	modules := idx.byBaseName[name]
	for _, m := range modules {
		if m.Service == context.Service &&
			m.Environment == context.Environment &&
			m.Region == context.Region {
			return m
		}
	}

	return nil
}
