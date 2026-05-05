package fsstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
)

// StoreDeps groups the internal store dependencies for explicit assembly.
type StoreDeps struct {
	Layout   PathLayout
	Metadata MetadataCodec
	Writer   ObjectWriter
}

// Store is a filesystem-backed blob store implementation.
//
// Concurrency: Store serializes write/delete operations on the same
// (namespace, key) pair via an in-process keyed mutex. This protects callers
// using a single Store within one process — the typical TerraCi scenario.
//
// Cross-process protection (multiple TerraCi processes sharing the same root
// directory) is best-effort: the underlying temp+rename sequence preserves
// per-file atomicity, but data and metadata are committed in two separate
// rename calls that can interleave with another process. Run separate
// TerraCi instances against distinct root_dir paths if strict cross-process
// isolation is required.
type Store struct {
	layout   PathLayout
	metadata MetadataCodec
	writer   ObjectWriter

	keyMu sync.Mutex
	locks map[string]*sync.Mutex
}

// New constructs a filesystem-backed blob store rooted at rootDir.
func New(rootDir string) *Store {
	layout := NewNestedPathLayout(rootDir)
	metadata := FileMetadataCodec{}
	return NewWithDeps(StoreDeps{
		Layout:   layout,
		Metadata: metadata,
		Writer:   NewFileObjectWriter(metadata, realClock{}),
	})
}

// NewWithDeps constructs a store from explicit internal dependencies.
func NewWithDeps(deps StoreDeps) *Store {
	return &Store{
		layout:   deps.Layout,
		metadata: deps.Metadata,
		writer:   deps.Writer,
		locks:    make(map[string]*sync.Mutex),
	}
}

// objectLock returns the keyed mutex for a (namespace, key) pair. Locks are
// allocated lazily and retained for the life of the Store to keep concurrent
// operations on the same key serialized.
func (s *Store) objectLock(namespace, key string) *sync.Mutex {
	s.keyMu.Lock()
	defer s.keyMu.Unlock()

	if s.locks == nil {
		s.locks = make(map[string]*sync.Mutex)
	}
	id := namespace + "\x00" + key
	mu, ok := s.locks[id]
	if !ok {
		mu = &sync.Mutex{}
		s.locks[id] = mu
	}
	return mu
}

// BlobStoreRootDir returns the root directory used for blob storage.
func (s *Store) BlobStoreRootDir() string {
	return s.layout.RootDir()
}

// DescribeBlobStore returns backend diagnostics for disk-backed blob storage.
func (s *Store) DescribeBlobStore() blobcache.Info {
	return blobcache.Info{
		Backend:                 "diskblob",
		Root:                    s.layout.RootDir(),
		SupportsList:            true,
		SupportsStream:          true,
		SupportsDeleteNamespace: true,
	}
}

// CheckBlobStore verifies that the configured root looks usable.
func (s *Store) CheckBlobStore(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("diskblob: health check failed: %w", err)
	}
	if err := ValidateRootDir(s.layout.RootDir()); err != nil {
		return fmt.Errorf("diskblob: health check failed: %w", err)
	}

	rootDir := s.layout.RootDir()
	if info, err := os.Stat(rootDir); err == nil && info.IsDir() {
		if _, err := os.ReadDir(rootDir); err != nil {
			return fmt.Errorf("diskblob: health check failed: read root_dir: %w", err)
		}
	}

	return nil
}

// Get loads a blob fully into memory.
func (s *Store) Get(_ context.Context, namespace, key string) (data []byte, ok bool, meta blobcache.Meta, err error) {
	var paths ObjectPaths
	paths, err = s.resolveObject(namespace, key)
	if err != nil {
		return nil, false, blobcache.Meta{}, fmt.Errorf("diskblob: resolve blob path: %w", err)
	}

	data, err = os.ReadFile(paths.DataPath)
	if os.IsNotExist(err) {
		return nil, false, blobcache.Meta{}, nil
	}
	if err != nil {
		return nil, false, blobcache.Meta{}, fmt.Errorf("diskblob: read blob: %w", err)
	}

	meta, err = s.metadata.Read(paths.MetaPath)
	if err != nil {
		return nil, false, blobcache.Meta{}, fmt.Errorf("diskblob: read blob metadata: %w", err)
	}

	return data, true, meta, nil
}

// Put stores a byte slice as a blob.
func (s *Store) Put(ctx context.Context, namespace, key string, value []byte, opts blobcache.PutOptions) (blobcache.Meta, error) {
	return s.PutStream(ctx, namespace, key, bytes.NewReader(value), opts)
}

// Open opens a streaming reader for a blob.
func (s *Store) Open(_ context.Context, namespace, key string) (io.ReadCloser, bool, blobcache.Meta, error) {
	paths, err := s.resolveObject(namespace, key)
	if err != nil {
		return nil, false, blobcache.Meta{}, fmt.Errorf("diskblob: resolve blob path: %w", err)
	}

	file, err := os.Open(paths.DataPath)
	if os.IsNotExist(err) {
		return nil, false, blobcache.Meta{}, nil
	}
	if err != nil {
		return nil, false, blobcache.Meta{}, fmt.Errorf("diskblob: open blob: %w", err)
	}

	meta, err := s.metadata.Read(paths.MetaPath)
	if err != nil {
		_ = file.Close()
		return nil, false, blobcache.Meta{}, fmt.Errorf("diskblob: read blob metadata: %w", err)
	}

	return file, true, meta, nil
}

// PutStream stores a streamed blob and metadata.
func (s *Store) PutStream(ctx context.Context, namespace, key string, r io.Reader, opts blobcache.PutOptions) (blobcache.Meta, error) {
	paths, err := s.resolveObject(namespace, key)
	if err != nil {
		return blobcache.Meta{}, fmt.Errorf("diskblob: resolve blob path: %w", err)
	}

	mu := s.objectLock(namespace, key)
	mu.Lock()
	defer mu.Unlock()

	meta, err := s.writer.Write(ctx, paths, r, opts)
	if err != nil {
		return blobcache.Meta{}, fmt.Errorf("diskblob: write blob data: %w", err)
	}

	return meta, nil
}

// Delete removes a single blob and its metadata sidecar.
func (s *Store) Delete(_ context.Context, namespace, key string) error {
	paths, err := s.resolveObject(namespace, key)
	if err != nil {
		return fmt.Errorf("diskblob: resolve blob path: %w", err)
	}

	mu := s.objectLock(namespace, key)
	mu.Lock()
	defer mu.Unlock()

	if err := os.Remove(paths.DataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("diskblob: delete blob data: %w", err)
	}
	if err := os.Remove(paths.MetaPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("diskblob: delete blob metadata: %w", err)
	}

	return nil
}

// DeleteNamespace removes all blobs in a namespace.
func (s *Store) DeleteNamespace(_ context.Context, namespace string) error {
	ns, err := parseNamespaceBoundary(namespace)
	if err != nil {
		return fmt.Errorf("diskblob: resolve namespace path: %w", err)
	}

	namespacePath, err := s.layout.ResolveNamespace(ns)
	if err != nil {
		return fmt.Errorf("diskblob: resolve namespace path: %w", err)
	}

	if err := os.RemoveAll(namespacePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("diskblob: delete namespace: %w", err)
	}

	return nil
}

// List returns all blobs stored under a namespace.
func (s *Store) List(_ context.Context, namespace string) ([]blobcache.Object, error) {
	ns, err := parseNamespaceBoundary(namespace)
	if err != nil {
		return nil, fmt.Errorf("diskblob: resolve namespace path: %w", err)
	}

	namespacePath, err := s.layout.ResolveNamespace(ns)
	if err != nil {
		return nil, fmt.Errorf("diskblob: resolve namespace path: %w", err)
	}

	var out []blobcache.Object
	err = filepath.WalkDir(namespacePath, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return fmt.Errorf("walk namespace: %w", walkErr)
		}
		if entry.IsDir() || strings.HasSuffix(current, metadataSuffix) {
			return nil
		}

		key, decodeErr := s.layout.DecodeListedObject(ns, current)
		if decodeErr != nil {
			return fmt.Errorf("decode listed object: %w", decodeErr)
		}

		meta, readErr := s.metadata.Read(current + metadataSuffix)
		if readErr != nil {
			return fmt.Errorf("read blob metadata: %w", readErr)
		}

		out = append(out, blobcache.Object{
			Key:  key.Value(),
			Meta: meta,
		})
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("diskblob: list namespace: %w", err)
	}

	return out, nil
}

func (s *Store) resolveObject(namespace, key string) (ObjectPaths, error) {
	ns, err := parseNamespaceBoundary(namespace)
	if err != nil {
		return ObjectPaths{}, err
	}

	blobKey, err := ParseBlobKey(key)
	if err != nil {
		return ObjectPaths{}, err
	}

	return s.layout.Resolve(ns, blobKey)
}
