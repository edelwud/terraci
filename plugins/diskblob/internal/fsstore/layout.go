package fsstore

import (
	"fmt"
	"path/filepath"
)

// PathLayout owns path construction and reverse decoding for the store.
type PathLayout interface {
	RootDir() string
	Resolve(namespace Namespace, key BlobKey) (ObjectPaths, error)
	ResolveNamespace(namespace Namespace) (string, error)
	DecodeListedObject(namespace Namespace, dataPath string) (BlobKey, error)
}

// NestedPathLayout preserves the v1 path-like namespace/key mapping.
type NestedPathLayout struct {
	rootDir string
}

// NewNestedPathLayout constructs the default disk layout rooted at rootDir.
func NewNestedPathLayout(rootDir string) NestedPathLayout {
	return NestedPathLayout{rootDir: rootDir}
}

// RootDir returns the layout root.
func (l NestedPathLayout) RootDir() string {
	return l.rootDir
}

// ResolveNamespace resolves the directory for a namespace.
func (l NestedPathLayout) ResolveNamespace(namespace Namespace) (string, error) {
	return namespaceRootPath(l.rootDir, namespace), nil
}

// Resolve resolves all filesystem paths for a blob.
func (l NestedPathLayout) Resolve(namespace Namespace, key BlobKey) (ObjectPaths, error) {
	namespacePath, err := l.ResolveNamespace(namespace)
	if err != nil {
		return ObjectPaths{}, fmt.Errorf("resolve namespace path: %w", err)
	}

	dataPath := filepath.Join(namespacePath, filepath.FromSlash(key.Value()))
	return ObjectPaths{
		NamespacePath: namespacePath,
		DataPath:      dataPath,
		MetaPath:      dataPath + metadataSuffix,
	}, nil
}

// DecodeListedObject converts a walked data path back into a blob key.
func (l NestedPathLayout) DecodeListedObject(namespace Namespace, dataPath string) (BlobKey, error) {
	namespacePath, err := l.ResolveNamespace(namespace)
	if err != nil {
		return BlobKey{}, fmt.Errorf("resolve namespace path: %w", err)
	}

	relPath, err := filepath.Rel(namespacePath, dataPath)
	if err != nil {
		return BlobKey{}, fmt.Errorf("decode listed object: %w", err)
	}

	return ParseBlobKey(filepath.ToSlash(relPath))
}
