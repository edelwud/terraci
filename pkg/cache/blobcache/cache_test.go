package blobcache

import (
	"context"
	"testing"
	"time"

	"github.com/edelwud/terraci/plugins/diskblob"
)

func TestCache_DefaultsAndAccessors(t *testing.T) {
	rootDir := t.TempDir()
	ttl := time.Hour
	cache := New(diskblob.NewStore(rootDir), "cost/pricing", ttl)

	if cache.Dir() != rootDir {
		t.Fatalf("Dir() = %q, want %q", cache.Dir(), rootDir)
	}
	if cache.TTL() != ttl {
		t.Fatalf("TTL() = %v, want %v", cache.TTL(), ttl)
	}
}

func TestCache_PutGetListAndCleanExpired(t *testing.T) {
	cache := New(diskblob.NewStore(t.TempDir()), "cost/pricing", time.Hour)
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
	cache := New(diskblob.NewStore(t.TempDir()), "cost/pricing", time.Hour)

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
