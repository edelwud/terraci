package fsstore

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const metadataSuffix = ".meta.json"

// Namespace is a validated blob namespace.
type Namespace struct {
	value string
	root  bool
}

// RootNamespace returns the store root namespace.
func RootNamespace() Namespace {
	return Namespace{root: true}
}

// ParseNamespace validates a non-root namespace.
func ParseNamespace(raw string) (Namespace, error) {
	value, err := validateRelativePath(raw)
	if err != nil {
		return Namespace{}, fmt.Errorf("invalid namespace %q: %w", raw, err)
	}
	return Namespace{value: value}, nil
}

// Value returns the normalized namespace path.
func (n Namespace) Value() string {
	return n.value
}

// IsRoot reports whether this namespace maps directly to the store root.
func (n Namespace) IsRoot() bool {
	return n.root
}

// BlobKey is a validated object key.
type BlobKey struct {
	value string
}

// ParseBlobKey validates a blob key.
func ParseBlobKey(raw string) (BlobKey, error) {
	value, err := validateRelativePath(raw)
	if err != nil {
		return BlobKey{}, fmt.Errorf("invalid key %q: %w", raw, err)
	}
	return BlobKey{value: value}, nil
}

// Value returns the normalized blob key.
func (k BlobKey) Value() string {
	return k.value
}

// ObjectPaths describes resolved filesystem locations for a blob.
type ObjectPaths struct {
	NamespacePath string
	DataPath      string
	MetaPath      string
}

func parseNamespaceBoundary(raw string) (Namespace, error) {
	if raw == "" {
		return RootNamespace(), nil
	}
	return ParseNamespace(raw)
}

func validateRelativePath(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("path must not be empty")
	}

	clean := path.Clean(strings.ReplaceAll(raw, "\\", "/"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) {
		return "", errors.New("path must be relative and must not escape root")
	}

	return clean, nil
}

func namespaceRootPath(rootDir string, namespace Namespace) string {
	if namespace.IsRoot() {
		return rootDir
	}
	return filepath.Join(rootDir, filepath.FromSlash(namespace.Value()))
}

// ValidateRootDir verifies that the configured root can be used by the backend.
func ValidateRootDir(rootDir string) error {
	if strings.TrimSpace(rootDir) == "" {
		return errors.New("root_dir must not be empty")
	}

	cleanRoot := filepath.Clean(rootDir)
	if _, err := filepath.Abs(cleanRoot); err != nil {
		return fmt.Errorf("resolve absolute root_dir: %w", err)
	}

	info, err := os.Stat(cleanRoot)
	switch {
	case err == nil:
		if !info.IsDir() {
			return errors.New("root_dir must point to a directory")
		}
		return nil
	case !os.IsNotExist(err):
		return fmt.Errorf("stat root_dir: %w", err)
	}

	parent := cleanRoot
	for {
		parent = filepath.Dir(parent)
		info, err = os.Stat(parent)
		switch {
		case err == nil:
			if !info.IsDir() {
				return errors.New("root_dir parent must be a directory")
			}
			return nil
		case !os.IsNotExist(err):
			return fmt.Errorf("stat root_dir parent: %w", err)
		}
		if parent == filepath.Dir(parent) {
			return errors.New("root_dir has no existing parent directory")
		}
	}
}
