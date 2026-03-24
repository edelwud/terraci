package discovery

import (
	"context"
	"path/filepath"
)

// Scanner discovers Terraform modules in a directory tree.
type Scanner struct {
	RootDir  string
	Segments []string
}

// NewScanner creates a Scanner from structure config values.
func NewScanner(rootDir string, segments []string) *Scanner {
	return &Scanner{
		RootDir:  rootDir,
		Segments: segments,
	}
}

// Scan walks the directory tree and returns all discovered Terraform modules.
func (s *Scanner) Scan(ctx context.Context) ([]*Module, error) {
	absRoot, err := filepath.Abs(s.RootDir)
	if err != nil {
		return nil, err
	}

	collector := &moduleCollector{
		ctx:      ctx,
		absRoot:  absRoot,
		segments: s.Segments,
		byID:     make(map[string]*Module),
	}

	if err := filepath.Walk(absRoot, collector.visit); err != nil {
		return nil, err
	}

	return collector.modules, nil
}
