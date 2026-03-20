package discovery

import "path/filepath"

// Scanner discovers Terraform modules in a directory tree.
type Scanner struct {
	RootDir  string
	MinDepth int
	MaxDepth int
	Segments []string
}

// NewScanner creates a Scanner from structure config values.
func NewScanner(rootDir string, minDepth, maxDepth int, segments []string) *Scanner {
	return &Scanner{
		RootDir:  rootDir,
		MinDepth: minDepth,
		MaxDepth: maxDepth,
		Segments: segments,
	}
}

// Scan walks the directory tree and returns all discovered Terraform modules.
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
