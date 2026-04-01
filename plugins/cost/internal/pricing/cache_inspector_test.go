package pricing

import (
	"context"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/plugins/diskblob"
)

func TestCacheInspector_EntriesAndOldestAge(t *testing.T) {
	store := diskblob.NewStore(t.TempDir())
	cache := blobcache.New(store, "", time.Hour)
	inspector := NewCacheInspector(cache)

	now := time.Now().Truncate(time.Second)
	expiresAt := now.Add(time.Hour)
	if _, err := cache.Put(context.Background(), "aws/AmazonEC2/us-east-1.json", []byte("ec2"), blobcache.PutOptions{
		ExpiresAt: &expiresAt,
	}); err != nil {
		t.Fatalf("Put(valid) error = %v", err)
	}
	if _, err := cache.Put(context.Background(), "invalid-key", []byte("skip"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put(invalid) error = %v", err)
	}

	entries := inspector.Entries(context.Background())
	if len(entries) != 1 {
		t.Fatalf("Entries() len = %d, want 1", len(entries))
	}
	if entries[0].Service != awsServiceEC2 {
		t.Fatalf("Entries()[0].Service = %q, want %q", entries[0].Service, awsServiceEC2)
	}
	if entries[0].Region != "us-east-1" {
		t.Fatalf("Entries()[0].Region = %q, want %q", entries[0].Region, "us-east-1")
	}
	if entries[0].ExpiresIn <= 0 {
		t.Fatalf("Entries()[0].ExpiresIn = %v, want > 0", entries[0].ExpiresIn)
	}
	if inspector.OldestAge(context.Background()) < 0 {
		t.Fatalf("OldestAge() = %v, want >= 0", inspector.OldestAge(context.Background()))
	}
}

func TestCacheInspector_DirAndTTL(t *testing.T) {
	root := t.TempDir()
	cache := blobcache.New(diskblob.NewStore(root), "", 2*time.Hour)
	inspector := NewCacheInspector(cache)

	if inspector.Dir() != root {
		t.Fatalf("Dir() = %q, want %q", inspector.Dir(), root)
	}
	if inspector.TTL() != 2*time.Hour {
		t.Fatalf("TTL() = %v, want %v", inspector.TTL(), 2*time.Hour)
	}
}

func TestCacheInspector_ParseKey(t *testing.T) {
	service, region, ok := parseCacheKey("aws/AmazonRDS/eu-west-1.json")
	if !ok {
		t.Fatal("parseCacheKey() ok = false, want true")
	}
	if service != awsServiceRDS {
		t.Fatalf("parseCacheKey().Service = %q, want %q", service, awsServiceRDS)
	}
	if region != "eu-west-1" {
		t.Fatalf("parseCacheKey().Region = %q, want %q", region, "eu-west-1")
	}

	if _, _, ok := parseCacheKey("broken"); ok {
		t.Fatal("parseCacheKey() ok = true for malformed key")
	}
}
