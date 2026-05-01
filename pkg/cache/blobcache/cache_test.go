package blobcache

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/plugin"
)

type memoryBlobStore struct {
	root    string
	objects map[string]memoryBlobObject
}

type memoryBlobObject struct {
	data []byte
	meta plugin.BlobMeta
}

func newMemoryBlobStore(root string) *memoryBlobStore {
	return &memoryBlobStore{
		root:    root,
		objects: make(map[string]memoryBlobObject),
	}
}

func (s *memoryBlobStore) BlobStoreRootDir() string {
	return s.root
}

func (s *memoryBlobStore) Get(_ context.Context, namespace, key string) (data []byte, ok bool, meta plugin.BlobMeta, err error) {
	object, ok := s.objects[namespace+"/"+key]
	if !ok {
		return nil, false, plugin.BlobMeta{}, nil
	}
	return append([]byte(nil), object.data...), true, cloneBlobMeta(object.meta), nil
}

func (s *memoryBlobStore) Put(_ context.Context, namespace, key string, value []byte, opts plugin.PutBlobOptions) (plugin.BlobMeta, error) {
	meta := plugin.BlobMeta{
		ContentType: opts.ContentType,
		UpdatedAt:   time.Now().UTC(),
		Size:        int64(len(value)),
		ExpiresAt:   cloneTimePtr(opts.ExpiresAt),
		Metadata:    cloneStringMap(opts.Metadata),
	}
	s.objects[namespace+"/"+key] = memoryBlobObject{
		data: append([]byte(nil), value...),
		meta: meta,
	}
	return cloneBlobMeta(meta), nil
}

func (s *memoryBlobStore) Open(context.Context, string, string) (io.ReadCloser, bool, plugin.BlobMeta, error) {
	return nil, false, plugin.BlobMeta{}, nil
}

func (s *memoryBlobStore) PutStream(context.Context, string, string, io.Reader, plugin.PutBlobOptions) (plugin.BlobMeta, error) {
	return plugin.BlobMeta{}, nil
}

func (s *memoryBlobStore) Delete(_ context.Context, namespace, key string) error {
	delete(s.objects, namespace+"/"+key)
	return nil
}

func (s *memoryBlobStore) DeleteNamespace(_ context.Context, namespace string) error {
	for scopedKey := range s.objects {
		if len(scopedKey) >= len(namespace)+1 && scopedKey[:len(namespace)+1] == namespace+"/" {
			delete(s.objects, scopedKey)
		}
	}
	return nil
}

func (s *memoryBlobStore) List(_ context.Context, namespace string) ([]plugin.BlobObject, error) {
	prefix := namespace + "/"
	objects := make([]plugin.BlobObject, 0, len(s.objects))
	for scopedKey, object := range s.objects {
		if len(scopedKey) < len(prefix) || scopedKey[:len(prefix)] != prefix {
			continue
		}
		objects = append(objects, plugin.BlobObject{
			Key:  scopedKey[len(prefix):],
			Meta: cloneBlobMeta(object.meta),
		})
	}
	return objects, nil
}

func TestCache_DefaultsAndAccessors(t *testing.T) {
	rootDir := t.TempDir()
	ttl := time.Hour
	cache := New(newMemoryBlobStore(rootDir), "cache/pricing", ttl)

	if cache.Dir() != rootDir {
		t.Fatalf("Dir() = %q, want %q", cache.Dir(), rootDir)
	}
	if cache.TTL() != ttl {
		t.Fatalf("TTL() = %v, want %v", cache.TTL(), ttl)
	}
}

func TestCache_PutGetListAndCleanExpired(t *testing.T) {
	cache := New(newMemoryBlobStore(t.TempDir()), "cache/pricing", time.Hour)
	expiresAt := time.Now().Add(-time.Minute).UTC()

	if _, err := cache.Put(context.Background(), "aws/AmazonEC2/us-east-1.json", []byte("payload"), PutOptions{
		ContentType: "application/json",
		ExpiresAt:   &expiresAt,
		Metadata: map[string]string{
			"kind": "pricing",
		},
	}); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	data, meta, ok, err := cache.Get(context.Background(), "aws/AmazonEC2/us-east-1.json")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok || string(data) != "payload" {
		t.Fatalf("Get() = (%q, %v), want (%q, true)", string(data), ok, "payload")
	}
	if meta.ContentType != "application/json" || meta.Metadata["kind"] != "pricing" {
		t.Fatalf("Get() meta = %+v, want preserved metadata", meta)
	}

	objects, err := cache.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("List() len = %d, want 1", len(objects))
	}
	if objects[0].Key != "aws/AmazonEC2/us-east-1.json" {
		t.Fatalf("List()[0].Key = %q, want expected key", objects[0].Key)
	}
	if objects[0].ExpiresIn >= 0 {
		t.Fatalf("List()[0].ExpiresIn = %v, want expired entry", objects[0].ExpiresIn)
	}

	if cleanErr := cache.CleanExpired(context.Background()); cleanErr != nil {
		t.Fatalf("CleanExpired() error = %v", cleanErr)
	}
	_, _, ok, err = cache.Get(context.Background(), "aws/AmazonEC2/us-east-1.json")
	if err != nil {
		t.Fatalf("Get() after cleanup error = %v", err)
	}
	if ok {
		t.Fatal("Get() after cleanup ok = true, want false")
	}
}

func TestCache_DeleteAndDeleteNamespace(t *testing.T) {
	cache := New(newMemoryBlobStore(t.TempDir()), "cache/pricing", time.Hour)

	if _, err := cache.Put(context.Background(), "one.json", []byte("one"), PutOptions{}); err != nil {
		t.Fatalf("Put(one) error = %v", err)
	}
	if _, err := cache.Put(context.Background(), "two.json", []byte("two"), PutOptions{}); err != nil {
		t.Fatalf("Put(two) error = %v", err)
	}

	if err := cache.Delete(context.Background(), "one.json"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, _, ok, err := cache.Get(context.Background(), "one.json"); err != nil || ok {
		t.Fatalf("Get() after delete = (%v, %v), want (nil, false)", err, ok)
	}

	if err := cache.DeleteNamespace(context.Background()); err != nil {
		t.Fatalf("DeleteNamespace() error = %v", err)
	}
	objects, err := cache.List(context.Background())
	if err != nil {
		t.Fatalf("List() after namespace delete error = %v", err)
	}
	if len(objects) != 0 {
		t.Fatalf("List() after namespace delete len = %d, want 0", len(objects))
	}
}
