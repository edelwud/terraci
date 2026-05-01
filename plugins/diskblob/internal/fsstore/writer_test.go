package fsstore

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type failingMetadataCodec struct{}

func (failingMetadataCodec) Read(string) (blobcache.Meta, error) {
	return blobcache.Meta{}, nil
}

func (failingMetadataCodec) Write(string, blobcache.Meta) error {
	return os.ErrPermission
}

func TestFileObjectWriterUsesClockAndLeavesNoTempFiles(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	now := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	writer := NewFileObjectWriter(FileMetadataCodec{}, fixedClock{now: now})

	paths := ObjectPaths{
		DataPath: filepath.Join(rootDir, "blob.bin"),
		MetaPath: filepath.Join(rootDir, "blob.bin.meta.json"),
	}

	meta, err := writer.Write(context.Background(), paths, strings.NewReader("payload"), blobcache.PutOptions{})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if !meta.UpdatedAt.Equal(now) {
		t.Fatalf("Write().UpdatedAt = %v, want %v", meta.UpdatedAt, now)
	}

	entries, err := os.ReadDir(rootDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ReadDir() len = %d, want 2", len(entries))
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "blob-") || strings.HasPrefix(entry.Name(), "blob-meta-") {
			t.Fatalf("unexpected temp file left behind: %s", entry.Name())
		}
	}
}

func TestStoreMetadataWriteFailureWrapped(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	layout := NewNestedPathLayout(rootDir)
	store := NewWithDeps(StoreDeps{
		Layout:   layout,
		Metadata: failingMetadataCodec{},
		Writer:   NewFileObjectWriter(failingMetadataCodec{}, fixedClock{now: time.Now().UTC()}),
	})

	_, err := store.PutStream(context.Background(), "cost/pricing", "aws/AmazonEC2/us-east-1.json", strings.NewReader("payload"), blobcache.PutOptions{})
	if err == nil {
		t.Fatal("PutStream() error = nil, want wrapped metadata failure")
	}
	if !strings.Contains(err.Error(), "diskblob: write blob data") || !strings.Contains(err.Error(), "write metadata") {
		t.Fatalf("PutStream() error = %q, want wrapped metadata write context", err)
	}
}

func TestStoreOpenCorruptMetadataWrapped(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	store := New(rootDir)
	if _, err := store.Put(context.Background(), "cost/pricing", "aws/AmazonEC2/us-east-1.json", []byte("payload"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(rootDir, "cost", "pricing", "aws", "AmazonEC2", "us-east-1.json.meta.json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r, ok, _, err := store.Open(context.Background(), "cost/pricing", "aws/AmazonEC2/us-east-1.json")
	if r != nil {
		_ = r.Close()
	}
	if ok {
		t.Fatal("Open() ok = true, want false on corrupt metadata")
	}
	if err == nil || !strings.Contains(err.Error(), "diskblob: read blob metadata") {
		t.Fatalf("Open() error = %v, want wrapped metadata read error", err)
	}
}

func TestDataWrittenByExistingLayoutRemainsReadable(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	dataPath := filepath.Join(rootDir, "cost", "pricing", "aws", "AmazonEC2", "us-east-1.json")
	if err := os.MkdirAll(filepath.Dir(dataPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(dataPath, []byte("legacy"), 0o600); err != nil {
		t.Fatalf("WriteFile(data) error = %v", err)
	}
	codec := FileMetadataCodec{}
	if err := codec.Write(dataPath+metadataSuffix, blobcache.Meta{Size: 6, UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("Write(meta) error = %v", err)
	}

	store := New(rootDir)
	data, ok, _, err := store.Get(context.Background(), "cost/pricing", "aws/AmazonEC2/us-east-1.json")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok || string(data) != "legacy" {
		t.Fatalf("Get() = (%q, %v), want (%q, true)", string(data), ok, "legacy")
	}
}

func TestStoreListReturnsOnlyBlobObjects(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	store := New(rootDir)
	if _, err := store.Put(context.Background(), "cost/pricing", "blob.bin", []byte("payload"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "cost", "pricing", "README.txt.meta.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile(meta-only) error = %v", err)
	}

	objects, err := store.List(context.Background(), "cost/pricing")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("List() len = %d, want 1", len(objects))
	}
	if objects[0].Key != "blob.bin" {
		t.Fatalf("List()[0].Key = %q, want blob.bin", objects[0].Key)
	}
}

func TestFileObjectWriterOpenRoundTrip(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	store := New(rootDir)
	if _, err := store.PutStream(context.Background(), "cost/pricing", "blob.bin", strings.NewReader("streamed"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("PutStream() error = %v", err)
	}

	reader, ok, _, err := store.Open(context.Background(), "cost/pricing", "blob.bin")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if !ok {
		t.Fatal("Open() ok = false, want true")
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(data) != "streamed" {
		t.Fatalf("Open() data = %q, want streamed", string(data))
	}
}

func TestNewWithDeps_EquivalentToNew(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	defaultStore := New(rootDir)
	customStore := NewWithDeps(StoreDeps{
		Layout:   NewNestedPathLayout(rootDir),
		Metadata: FileMetadataCodec{},
		Writer:   NewFileObjectWriter(FileMetadataCodec{}, realClock{}),
	})

	if defaultStore.DescribeBlobStore() != customStore.DescribeBlobStore() {
		t.Fatalf("DescribeBlobStore() mismatch: default=%+v custom=%+v", defaultStore.DescribeBlobStore(), customStore.DescribeBlobStore())
	}
}
