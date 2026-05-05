package discovery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// moduleCollector accumulates discovered modules during directory walk.
type moduleCollector struct {
	ctx      context.Context
	absRoot  string
	segments []string
	modules  []*Module
	byID     map[string]*Module
}

func (c *moduleCollector) visit(path string, info os.FileInfo, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}

	if err := c.ctx.Err(); err != nil {
		return err
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

	if len(parts) < len(c.segments) {
		return nil
	}

	c.registerModule(parts, path)
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
	segCount := len(c.segments)
	if len(parts) <= segCount {
		return NewModule(c.segments, parts, absPath, relPath)
	}

	mod := NewModule(c.segments, parts[:segCount], absPath, relPath)
	mod.SetComponent("submodule", filepath.Join(parts[segCount:]...))
	c.linkParent(mod, parts)
	return mod
}

func (c *moduleCollector) linkParent(mod *Module, parts []string) {
	parentRelPath := filepath.Join(parts[:len(parts)-1]...)
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

// containsTerraformFiles checks if a directory contains .tf files. Reads the
// directory entries directly instead of going through filepath.Glob: the
// scanner is invoked once per directory in the walk, so on a 10K-directory
// repo the difference is ~10K syscalls (Glob does Lstat-of-parent +
// pattern parse + ReadDir; this path is only ReadDir).
func containsTerraformFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".tf") {
			return true
		}
	}
	return false
}
