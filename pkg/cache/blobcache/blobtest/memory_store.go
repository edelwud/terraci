package blobtest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
)

// NewMemoryStore returns an in-memory blob store suitable for tests.
func NewMemoryStore(root string) blobcache.Store {
	return &memoryStore{
		root:    root,
		objects: make(map[string]memoryObject),
	}
}

type memoryStore struct {
	root    string
	mu      sync.RWMutex
	objects map[string]memoryObject
}

type memoryObject struct {
	data []byte
	meta blobcache.Meta
}

func (s *memoryStore) BlobStoreRootDir() string {
	return s.root
}

func (s *memoryStore) Get(_ context.Context, namespace, key string) (data []byte, ok bool, meta blobcache.Meta, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	object, ok := s.objects[scopedKey(namespace, key)]
	if !ok {
		return nil, false, blobcache.Meta{}, nil
	}
	return append([]byte(nil), object.data...), true, cloneMeta(object.meta), nil
}

func (s *memoryStore) Put(_ context.Context, namespace, key string, value []byte, opts blobcache.PutOptions) (blobcache.Meta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta := blobcache.Meta{
		Size:        int64(len(value)),
		UpdatedAt:   time.Now().UTC(),
		ExpiresAt:   cloneTimePtr(opts.ExpiresAt),
		ETag:        etag(value),
		ContentType: opts.ContentType,
		Metadata:    cloneStringMap(opts.Metadata),
	}
	s.objects[scopedKey(namespace, key)] = memoryObject{
		data: append([]byte(nil), value...),
		meta: cloneMeta(meta),
	}
	return cloneMeta(meta), nil
}

func (s *memoryStore) Open(ctx context.Context, namespace, key string) (io.ReadCloser, bool, blobcache.Meta, error) {
	data, ok, meta, err := s.Get(ctx, namespace, key)
	if err != nil || !ok {
		return nil, ok, meta, err
	}
	return io.NopCloser(bytes.NewReader(data)), true, meta, nil
}

func (s *memoryStore) PutStream(ctx context.Context, namespace, key string, r io.Reader, opts blobcache.PutOptions) (blobcache.Meta, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return blobcache.Meta{}, err
	}
	return s.Put(ctx, namespace, key, data, opts)
}

func (s *memoryStore) Delete(_ context.Context, namespace, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.objects, scopedKey(namespace, key))
	return nil
}

func (s *memoryStore) DeleteNamespace(_ context.Context, namespace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := namespace + "/"
	for scoped := range s.objects {
		if strings.HasPrefix(scoped, prefix) {
			delete(s.objects, scoped)
		}
	}
	return nil
}

func (s *memoryStore) List(_ context.Context, namespace string) ([]blobcache.Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prefix := namespace + "/"
	objects := make([]blobcache.Object, 0, len(s.objects))
	for scoped, object := range s.objects {
		if !strings.HasPrefix(scoped, prefix) {
			continue
		}
		objects = append(objects, blobcache.Object{
			Key:  strings.TrimPrefix(scoped, prefix),
			Meta: cloneMeta(object.meta),
		})
	}
	return objects, nil
}

func scopedKey(namespace, key string) string {
	return namespace + "/" + key
}

func cloneMeta(meta blobcache.Meta) blobcache.Meta {
	meta.ExpiresAt = cloneTimePtr(meta.ExpiresAt)
	meta.Metadata = cloneStringMap(meta.Metadata)
	return meta
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	copied := *t
	return &copied
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}

func etag(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
