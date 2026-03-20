package discovery

import (
	"os"
	"path/filepath"
	"strings"
)

// moduleCollector accumulates discovered modules during directory walk.
type moduleCollector struct {
	absRoot  string
	minDepth int
	maxDepth int
	segments []string
	modules  []*Module
	byID     map[string]*Module
}

func (c *moduleCollector) visit(path string, info os.FileInfo, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}

	if shouldSkipDir(info) {
		return filepath.SkipDir
	}

	if !isTerraformDir(info, path) {
		return nil
	}

	parts, ok := c.parseRelPath(path)
	if !ok {
		return nil
	}

	depth := len(parts)
	if depth < c.minDepth {
		return nil
	}
	if depth > c.maxDepth {
		return filepath.SkipDir
	}

	c.registerModule(parts, path)

	if depth >= c.maxDepth {
		return filepath.SkipDir
	}
	return nil
}

func (c *moduleCollector) parseRelPath(path string) ([]string, bool) {
	relPath, err := filepath.Rel(c.absRoot, path)
	if err != nil || relPath == "." {
		return nil, false
	}
	return strings.Split(relPath, string(os.PathSeparator)), true
}

func (c *moduleCollector) registerModule(parts []string, absPath string) {
	relPath := filepath.Join(parts...)
	mod := c.buildModule(parts, absPath, relPath)
	c.byID[mod.ID()] = mod
	c.modules = append(c.modules, mod)
}

func (c *moduleCollector) buildModule(parts []string, absPath, relPath string) *Module {
	isSubmodule := len(parts) > len(c.segments)

	if !isSubmodule {
		return NewModule(c.segments, parts, absPath, relPath)
	}

	mod := NewModule(c.segments, parts[:len(c.segments)], absPath, relPath)
	mod.SetComponent("submodule", parts[len(c.segments)])
	c.linkParent(mod, parts)
	return mod
}

func (c *moduleCollector) linkParent(mod *Module, parts []string) {
	parentRelPath := filepath.Join(parts[:len(c.segments)]...)
	if parent, ok := c.byID[parentRelPath]; ok {
		mod.Parent = parent
		parent.Children = append(parent.Children, mod)
	}
}

func shouldSkipDir(info os.FileInfo) bool {
	return info.IsDir() && strings.HasPrefix(info.Name(), ".")
}

func isTerraformDir(info os.FileInfo, path string) bool {
	return info.IsDir() && containsTerraformFiles(path)
}

// containsTerraformFiles checks if a directory contains .tf files.
func containsTerraformFiles(dir string) bool {
	matches, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return false
	}
	return len(matches) > 0
}
