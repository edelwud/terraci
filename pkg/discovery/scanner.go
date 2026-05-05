package discovery

import (
	"context"
	"path/filepath"
	"strings"
)

// Scanner discovers Terraform modules in a directory tree.
type Scanner struct {
	RootDir  string
	Segments []string

	// LibraryPaths are project-relative directories whose descendants are
	// treated as library/shared modules (Module.IsLibrary=true). Paths are
	// sanity-cleaned (trim, drop empty, drop absolute, dedup, slash-normalize)
	// in NewScanner.
	LibraryPaths []string
}

// NewScanner creates a Scanner from structure config values and optional
// library module roots. Library paths are normalized to forward-slash, project-
// relative form; absolute paths and ".." escapes are dropped silently.
func NewScanner(rootDir string, segments []string, libraryPaths ...string) *Scanner {
	return &Scanner{
		RootDir:      rootDir,
		Segments:     segments,
		LibraryPaths: cleanLibraryPaths(libraryPaths),
	}
}

// Scan walks the directory tree and returns all discovered Terraform modules.
func (s *Scanner) Scan(ctx context.Context) ([]*Module, error) {
	absRoot, err := filepath.Abs(s.RootDir)
	if err != nil {
		return nil, err
	}

	collector := &moduleCollector{
		ctx:          ctx,
		absRoot:      absRoot,
		segments:     s.Segments,
		libraryPaths: s.LibraryPaths,
		byID:         make(map[string]*Module),
	}

	if err := filepath.Walk(absRoot, collector.visit); err != nil {
		return nil, err
	}

	return collector.modules, nil
}

// cleanLibraryPaths normalizes a list of library roots: trims spaces, drops
// empty/absolute/escaping entries, converts to forward slashes, and dedups
// while preserving input order.
func cleanLibraryPaths(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		if filepath.IsAbs(p) {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(p))
		if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
