package policy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edelwud/terraci/pkg/config"
)

// Source is an interface for policy sources
type Source interface {
	// Pull downloads policies to the destination directory
	Pull(ctx context.Context, dest string) error

	// String returns a human-readable description
	String() string
}

// NewSource creates a Source from a PolicySource config
func NewSource(cfg config.PolicySource) (Source, error) {
	switch cfg.Type() {
	case "path":
		return &PathSource{Path: cfg.Path}, nil
	case "git":
		return &GitSource{URL: cfg.Git, Ref: cfg.Ref}, nil
	case "oci":
		return &OCISource{URL: cfg.OCI}, nil
	default:
		return nil, fmt.Errorf("unknown policy source type")
	}
}

// Puller handles pulling policies from multiple sources
type Puller struct {
	sources  []Source
	cacheDir string
}

// NewPuller creates a new policy puller
func NewPuller(cfg *config.PolicyConfig, rootDir string) (*Puller, error) {
	if cfg == nil {
		return nil, fmt.Errorf("policy config is nil")
	}

	cacheDir := cfg.CacheDir
	if cacheDir == "" {
		cacheDir = ".terraci/policies"
	}

	// Make cache dir absolute if relative
	if !filepath.IsAbs(cacheDir) {
		cacheDir = filepath.Join(rootDir, cacheDir)
	}

	sources := make([]Source, 0, len(cfg.Sources))
	for _, srcCfg := range cfg.Sources {
		src, err := NewSource(srcCfg)
		if err != nil {
			return nil, fmt.Errorf("invalid source: %w", err)
		}

		// Resolve relative paths for path sources
		if ps, ok := src.(*PathSource); ok && !filepath.IsAbs(ps.Path) {
			ps.Path = filepath.Join(rootDir, ps.Path)
		}

		sources = append(sources, src)
	}

	return &Puller{
		sources:  sources,
		cacheDir: cacheDir,
	}, nil
}

// Pull downloads all policies to the cache directory
// Returns the list of directories containing policies
func (p *Puller) Pull(ctx context.Context) ([]string, error) {
	// Create cache directory
	if err := os.MkdirAll(p.cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	dirs := make([]string, 0, len(p.sources))

	for i, src := range p.sources {
		// Each source gets its own subdirectory
		dest := filepath.Join(p.cacheDir, fmt.Sprintf("source-%d", i))

		// For path sources, just use the path directly
		if ps, ok := src.(*PathSource); ok {
			dirs = append(dirs, ps.Path)
			continue
		}

		// Pull to cache
		if err := src.Pull(ctx, dest); err != nil {
			return nil, fmt.Errorf("failed to pull from %s: %w", src, err)
		}

		dirs = append(dirs, dest)
	}

	return dirs, nil
}

// CacheDir returns the cache directory path
func (p *Puller) CacheDir() string {
	return p.cacheDir
}
