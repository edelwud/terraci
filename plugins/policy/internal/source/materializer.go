package source

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

type Source interface {
	Materialize(ctx context.Context, dest string) (string, error)
	String() string
}

type Materializer struct {
	sources  []Source
	cacheDir string
	rootDir  string
}

func NewMaterializer(cfg *policyengine.Config, rootDir, serviceDir string) (*Materializer, error) {
	if cfg == nil {
		return nil, errors.New("policy config is nil")
	}

	cacheDir := resolveCacheDir(rootDir, serviceDir, cfg.SourceCacheDir)

	sources := make([]Source, 0, len(cfg.Sources))
	for i, sourceConfig := range cfg.Sources {
		src, err := NewSource(sourceConfig, rootDir)
		if err != nil {
			return nil, fmt.Errorf("sources[%d]: %w", i, err)
		}
		sources = append(sources, src)
	}

	return &Materializer{sources: sources, cacheDir: cacheDir, rootDir: rootDir}, nil
}

func NewSource(cfg policyengine.SourceConfig, rootDir string) (Source, error) {
	switch cfg.Type {
	case policyengine.SourceTypePath:
		path := cfg.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(rootDir, path)
		}
		return &PathSource{Path: path}, nil
	case policyengine.SourceTypeGit:
		return &GitSource{URL: cfg.URL, Ref: cfg.Ref}, nil
	case policyengine.SourceTypeOCI:
		return &OCISource{URL: cfg.URL}, nil
	default:
		return nil, fmt.Errorf("unsupported policy source type %q", cfg.Type)
	}
}

func (m *Materializer) Materialize(ctx context.Context, cacheDirOverride string) ([]string, error) {
	if m == nil {
		return nil, errors.New("policy source materializer is nil")
	}
	cacheDir := m.CacheDir(cacheDirOverride)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create policy cache dir: %w", err)
	}

	dirs := make([]string, 0, len(m.sources))
	for i, src := range m.sources {
		dest := filepath.Join(cacheDir, sourceDirName(i, src))
		dir, err := src.Materialize(ctx, dest)
		if err != nil {
			return nil, fmt.Errorf("materialize %s: %w", src, err)
		}
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

func (m *Materializer) CacheDir(cacheDirOverride string) string {
	if m == nil {
		return ""
	}
	if cacheDirOverride != "" {
		return resolveCacheDir(m.rootDir, "", cacheDirOverride)
	}
	return m.cacheDir
}

func resolveCacheDir(rootDir, serviceDir, configured string) string {
	cacheDir := configured
	if cacheDir == "" {
		if serviceDir == "" {
			serviceDir = ".terraci"
		}
		cacheDir = filepath.Join(serviceDir, "policies")
	}
	if !filepath.IsAbs(cacheDir) {
		cacheDir = filepath.Join(rootDir, cacheDir)
	}
	return cacheDir
}

func sourceDirName(index int, src Source) string {
	kind := "source"
	switch src.(type) {
	case *PathSource:
		kind = "path"
	case *GitSource:
		kind = "git"
	case *OCISource:
		kind = "oci"
	}
	return fmt.Sprintf("source-%02d-%s", index, kind)
}
