package discovery

import (
	"os"
	"path/filepath"
	"strings"
)

// moduleCollector accumulates discovered modules during directory walk
type moduleCollector struct {
	absRoot  string
	minDepth int
	maxDepth int
	segments []string
	modules  []*Module
	byID     map[string]*Module
}

func (c *moduleCollector) visit(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
		return filepath.SkipDir
	}

	if !info.IsDir() || !containsTerraformFiles(path) {
		return nil
	}

	relPath, err := filepath.Rel(c.absRoot, path)
	if err != nil {
		return err
	}
	if relPath == "." {
		return nil
	}

	parts := strings.Split(relPath, string(os.PathSeparator))
	depth := len(parts)

	switch {
	case depth < c.minDepth:
		return nil
	case depth > c.maxDepth:
		return filepath.SkipDir
	}

	module := c.buildModule(parts, path, relPath)
	c.byID[module.ID()] = module
	c.modules = append(c.modules, module)

	if depth >= c.maxDepth {
		return filepath.SkipDir
	}
	return nil
}

func (c *moduleCollector) buildModule(parts []string, path, relPath string) *Module {
	if len(parts) <= len(c.segments) {
		// Base module
		return NewModule(c.segments, parts, path, relPath)
	}
	// Submodule: base segments + extra level
	mod := NewModule(c.segments, parts[:len(c.segments)], path, relPath)
	mod.SetComponent("submodule", parts[len(c.segments)])
	// Link parent
	parentRelPath := filepath.Join(parts[:len(c.segments)]...)
	if parent, ok := c.byID[parentRelPath]; ok {
		mod.Parent = parent
		parent.Children = append(parent.Children, mod)
	}
	return mod
}

// containsTerraformFiles checks if a directory contains .tf files
func containsTerraformFiles(dir string) bool {
	matches, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return false
	}
	return len(matches) > 0
}
