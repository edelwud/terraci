package plugintest

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
)

// BlobStoreFactory creates a store instance bound to the provided backing id.
// Backends may interpret the backing id as a root directory, prefix, or another
// stable locator that allows creating another instance over the same storage.
type BlobStoreFactory func(tb testing.TB, backingID string) blobcache.Store

// BlobStoreContractSuite configures the shared blob-store contract checks.
type BlobStoreContractSuite struct {
	Factory      BlobStoreFactory
	Namespace    string
	AltNamespace string
	Key          string
	AltKey       string
	Payload      []byte
	StreamData   string
}

type blobStoreContractDefaults struct {
	namespace    string
	altNamespace string
	key          string
	altKey       string
	payload      []byte
	streamData   string
}

// RunBlobStoreContractSuite executes the shared blob-store backend contract checks.
func RunBlobStoreContractSuite(t *testing.T, suite BlobStoreContractSuite) {
	t.Helper()

	if suite.Factory == nil {
		t.Fatal("BlobStoreContractSuite.Factory must not be nil")
	}

	d := blobStoreSuiteDefaults(suite)

	t.Run("PutGet", func(t *testing.T) { runBlobStorePutGet(t, suite, d) })
	t.Run("PutStreamOpen", func(t *testing.T) { runBlobStorePutStreamOpen(t, suite, d) })
	t.Run("NamespaceIsolation", func(t *testing.T) { runBlobStoreNamespaceIsolation(t, suite, d) })
	t.Run("Delete", func(t *testing.T) { runBlobStoreDelete(t, suite, d) })
	t.Run("DeleteNamespace", func(t *testing.T) { runBlobStoreDeleteNamespace(t, suite, d) })
	t.Run("PersistenceAcrossInstances", func(t *testing.T) { runBlobStorePersistence(t, suite, d) })
	t.Run("MissingObject", func(t *testing.T) { runBlobStoreMissingObject(t, suite, d) })
}

func blobStoreSuiteDefaults(suite BlobStoreContractSuite) blobStoreContractDefaults {
	d := blobStoreContractDefaults{
		namespace:    suite.Namespace,
		altNamespace: suite.AltNamespace,
		key:          suite.Key,
		altKey:       suite.AltKey,
		payload:      suite.Payload,
		streamData:   suite.StreamData,
	}
	if d.namespace == "" {
		d.namespace = "contract/default"
	}
	if d.altNamespace == "" {
		d.altNamespace = "contract/alt"
	}
	if d.key == "" {
		d.key = "nested/blob.bin"
	}
	if d.altKey == "" {
		d.altKey = "other/blob.bin"
	}
	if len(d.payload) == 0 {
		d.payload = []byte("payload")
	}
	if d.streamData == "" {
		d.streamData = "stream-payload"
	}
	return d
}

func runBlobStorePutGet(t *testing.T, suite BlobStoreContractSuite, d blobStoreContractDefaults) {
	t.Helper()
	store := suite.Factory(t, t.TempDir())
	expiresAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	meta, err := store.Put(context.Background(), d.namespace, d.key, d.payload, blobcache.PutOptions{
		ContentType: "application/octet-stream",
		ExpiresAt:   &expiresAt,
		Metadata: map[string]string{
			"kind": "contract",
		},
	})
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	got, ok, readMeta, err := store.Get(context.Background(), d.namespace, d.key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if !bytes.Equal(got, d.payload) {
		t.Fatalf("Get() = %q, want %q", string(got), string(d.payload))
	}
	if meta.Size != int64(len(d.payload)) || readMeta.Size != meta.Size {
		t.Fatalf("metadata size mismatch: put=%d get=%d", meta.Size, readMeta.Size)
	}
	if readMeta.ContentType != "application/octet-stream" {
		t.Fatalf("Get().ContentType = %q, want application/octet-stream", readMeta.ContentType)
	}
	if readMeta.ExpiresAt == nil || !readMeta.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("Get().ExpiresAt = %v, want %v", readMeta.ExpiresAt, expiresAt)
	}
	if readMeta.Metadata["kind"] != "contract" {
		t.Fatalf("Get().Metadata = %+v, want contract marker", readMeta.Metadata)
	}
}

func runBlobStorePutStreamOpen(t *testing.T, suite BlobStoreContractSuite, d blobStoreContractDefaults) {
	t.Helper()
	store := suite.Factory(t, t.TempDir())
	if _, err := store.PutStream(context.Background(), d.namespace, d.key, bytes.NewBufferString(d.streamData), blobcache.PutOptions{}); err != nil {
		t.Fatalf("PutStream() error = %v", err)
	}

	reader, ok, meta, err := store.Open(context.Background(), d.namespace, d.key)
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
	if string(data) != d.streamData {
		t.Fatalf("Open() data = %q, want %q", string(data), d.streamData)
	}
	if meta.Size != int64(len(d.streamData)) {
		t.Fatalf("Open().Size = %d, want %d", meta.Size, len(d.streamData))
	}
}

func runBlobStoreNamespaceIsolation(t *testing.T, suite BlobStoreContractSuite, d blobStoreContractDefaults) {
	t.Helper()
	store := suite.Factory(t, t.TempDir())
	if _, err := store.Put(context.Background(), d.namespace, d.key, d.payload, blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put(primary) error = %v", err)
	}
	if _, err := store.Put(context.Background(), d.altNamespace, d.altKey, []byte("other"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put(alt) error = %v", err)
	}

	objects, err := store.List(context.Background(), d.namespace)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(objects) != 1 || objects[0].Key != d.key {
		t.Fatalf("List() = %+v, want one object with key %q", objects, d.key)
	}
}

func runBlobStoreDelete(t *testing.T, suite BlobStoreContractSuite, d blobStoreContractDefaults) {
	t.Helper()
	store := suite.Factory(t, t.TempDir())
	if _, err := store.Put(context.Background(), d.namespace, d.key, d.payload, blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if err := store.Delete(context.Background(), d.namespace, d.key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	got, ok, _, err := store.Get(context.Background(), d.namespace, d.key)
	if err != nil {
		t.Fatalf("Get() after delete error = %v", err)
	}
	if ok || got != nil {
		t.Fatalf("Get() after delete = (%q, %v), want (nil, false)", string(got), ok)
	}
}

func runBlobStoreDeleteNamespace(t *testing.T, suite BlobStoreContractSuite, d blobStoreContractDefaults) {
	t.Helper()
	store := suite.Factory(t, t.TempDir())
	if _, err := store.Put(context.Background(), d.namespace, d.key, d.payload, blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put(primary) error = %v", err)
	}
	if _, err := store.Put(context.Background(), d.namespace, d.altKey, []byte("other"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put(secondary) error = %v", err)
	}

	if err := store.DeleteNamespace(context.Background(), d.namespace); err != nil {
		t.Fatalf("DeleteNamespace() error = %v", err)
	}

	objects, err := store.List(context.Background(), d.namespace)
	if err != nil {
		t.Fatalf("List() after namespace delete error = %v", err)
	}
	if len(objects) != 0 {
		t.Fatalf("List() after namespace delete len = %d, want 0", len(objects))
	}
}

func runBlobStorePersistence(t *testing.T, suite BlobStoreContractSuite, d blobStoreContractDefaults) {
	t.Helper()
	backingID := t.TempDir()
	first := suite.Factory(t, backingID)
	if _, err := first.Put(context.Background(), d.namespace, d.key, d.payload, blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	second := suite.Factory(t, backingID)
	got, ok, _, err := second.Get(context.Background(), d.namespace, d.key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok || !bytes.Equal(got, d.payload) {
		t.Fatalf("Get() = (%q, %v), want (%q, true)", string(got), ok, string(d.payload))
	}
}

func runBlobStoreMissingObject(t *testing.T, suite BlobStoreContractSuite, d blobStoreContractDefaults) {
	t.Helper()
	store := suite.Factory(t, t.TempDir())
	got, ok, meta, err := store.Get(context.Background(), d.namespace, d.key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok || got != nil {
		t.Fatalf("Get() = (%v, %v), want (nil, false)", got, ok)
	}
	if meta.Size != 0 || !meta.UpdatedAt.IsZero() || meta.ExpiresAt != nil || meta.ETag != "" || meta.ContentType != "" || len(meta.Metadata) != 0 {
		t.Fatalf("Get() meta = %+v, want zero meta", meta)
	}
}
