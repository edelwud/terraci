// Package contracttest provides a shared test suite that validates the
// blobcache.Store contract. Backend authors (diskblob, future s3blob,
// gcsblob, etc.) call RunStoreContractTests with a factory that returns a
// fresh, empty Store rooted at a temp location; the suite then exercises
// the contractually required behavior — round-trip, ETag determinism,
// namespace isolation, concurrent put on the same key, list-after-delete,
// and traversal rejection.
//
// The suite is intentionally factory-based so backends with non-trivial
// setup (S3 buckets, GCS prefixes, redis prefixes) can plug in their own
// teardown hook via the test's t.Cleanup.
package contracttest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
)

// StoreFactory returns a freshly initialized, empty blobcache.Store for the
// duration of a single subtest. Implementations should register cleanup hooks
// via t.Cleanup so each subtest starts from a clean state.
type StoreFactory func(t *testing.T) blobcache.Store

// RunStoreContractTests exercises every contractual guarantee of
// blobcache.Store. Call this from a backend's _test.go:
//
//	func TestStoreContract(t *testing.T) {
//	    contracttest.RunStoreContractTests(t, func(t *testing.T) blobcache.Store {
//	        return fsstore.New(t.TempDir())
//	    })
//	}
//
// Each contract assertion lives in its own helper to keep cyclomatic
// complexity low and make individual assertions easy to skip / extend in
// downstream backends. Add new assertions by appending another helper here
// and a corresponding t.Run line below.
func RunStoreContractTests(t *testing.T, factory StoreFactory) {
	t.Helper()

	t.Run("RoundTrip", func(t *testing.T) { testRoundTrip(t, factory) })
	t.Run("ETagMatchesSHA256", func(t *testing.T) { testETagSHA256(t, factory) })
	t.Run("NamespaceIsolation", func(t *testing.T) { testNamespaceIsolation(t, factory) })
	t.Run("GetMissingReturnsFalse", func(t *testing.T) { testGetMissing(t, factory) })
	t.Run("ListAfterDelete", func(t *testing.T) { testListAfterDelete(t, factory) })
	t.Run("ConcurrentPutSameKey", func(t *testing.T) { testConcurrentPut(t, factory) })
	t.Run("StreamRoundTrip", func(t *testing.T) { testStreamRoundTrip(t, factory) })
	t.Run("PathTraversalRejected", func(t *testing.T) { testPathTraversalRejected(t, factory) })
	t.Run("DeleteNamespaceClearsAllKeys", func(t *testing.T) { testDeleteNamespace(t, factory) })
}

func testRoundTrip(t *testing.T, factory StoreFactory) {
	t.Helper()
	t.Parallel()
	store := factory(t)
	ctx := context.Background()

	_, err := store.Put(ctx, "ns", "key", []byte("hello"), blobcache.PutOptions{ContentType: "text/plain"})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	data, ok, meta, err := store.Get(ctx, "ns", "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("Get ok=false, want true after Put")
	}
	if string(data) != "hello" {
		t.Fatalf("Get data = %q, want %q", string(data), "hello")
	}
	if meta.Size != int64(len("hello")) {
		t.Errorf("meta.Size = %d, want %d", meta.Size, len("hello"))
	}
	if meta.ContentType != "text/plain" {
		t.Errorf("meta.ContentType = %q, want text/plain", meta.ContentType)
	}
}

func testETagSHA256(t *testing.T, factory StoreFactory) {
	t.Helper()
	t.Parallel()
	store := factory(t)
	ctx := context.Background()

	payload := []byte("etag-payload-content")
	if _, err := store.Put(ctx, "ns", "key", payload, blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	_, _, meta, err := store.Get(ctx, "ns", "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	hash := sha256.Sum256(payload)
	want := hex.EncodeToString(hash[:])
	if meta.ETag != want {
		t.Errorf("meta.ETag = %q, want sha256 %q", meta.ETag, want)
	}
}

func testNamespaceIsolation(t *testing.T, factory StoreFactory) {
	t.Helper()
	t.Parallel()
	store := factory(t)
	ctx := context.Background()

	if _, err := store.Put(ctx, "ns-a", "shared", []byte("A"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put ns-a: %v", err)
	}
	if _, err := store.Put(ctx, "ns-b", "shared", []byte("B"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put ns-b: %v", err)
	}

	dataA, _, _, err := store.Get(ctx, "ns-a", "shared")
	if err != nil {
		t.Fatalf("Get ns-a: %v", err)
	}
	dataB, _, _, err := store.Get(ctx, "ns-b", "shared")
	if err != nil {
		t.Fatalf("Get ns-b: %v", err)
	}
	if string(dataA) != "A" || string(dataB) != "B" {
		t.Errorf("namespace isolation broken: ns-a=%q ns-b=%q", string(dataA), string(dataB))
	}
}

func testGetMissing(t *testing.T, factory StoreFactory) {
	t.Helper()
	t.Parallel()
	store := factory(t)
	ctx := context.Background()

	_, ok, _, err := store.Get(ctx, "ns", "absent")
	if err != nil {
		t.Fatalf("Get on missing key returned error: %v (want nil)", err)
	}
	if ok {
		t.Error("Get ok=true on missing key")
	}
}

func testListAfterDelete(t *testing.T, factory StoreFactory) {
	t.Helper()
	t.Parallel()
	store := factory(t)
	ctx := context.Background()

	if _, err := store.Put(ctx, "ns", "k1", []byte("1"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put k1: %v", err)
	}
	if _, err := store.Put(ctx, "ns", "k2", []byte("2"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put k2: %v", err)
	}

	if err := store.Delete(ctx, "ns", "k1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	objects, err := store.List(ctx, "ns")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(objects) != 1 || objects[0].Key != "k2" {
		t.Fatalf("after Delete(k1), List = %v, want exactly [k2]", objects)
	}
}

func testConcurrentPut(t *testing.T, factory StoreFactory) {
	t.Helper()
	t.Parallel()
	store := factory(t)
	ctx := context.Background()

	const writers = 8
	const iterations = 16

	var wg sync.WaitGroup
	for i := range writers {
		wg.Go(func() {
			for j := range iterations {
				payload := fmt.Sprintf("writer-%d-iter-%d", i, j)
				if _, err := store.Put(ctx, "ns", "shared", []byte(payload), blobcache.PutOptions{}); err != nil {
					t.Errorf("Put: %v", err)
					return
				}
			}
		})
	}
	wg.Wait()

	// Final state: meta.ETag must be sha256(data). If concurrent writers
	// could interleave data and metadata commits, we'd see a mismatch.
	data, ok, meta, err := store.Get(ctx, "ns", "shared")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("Get ok=false after concurrent puts")
	}
	hash := sha256.Sum256(data)
	want := hex.EncodeToString(hash[:])
	if meta.ETag != want {
		t.Errorf("post-concurrent ETag mismatch: meta=%q sha256=%q", meta.ETag, want)
	}
	if meta.Size != int64(len(data)) {
		t.Errorf("post-concurrent Size mismatch: meta=%d len=%d", meta.Size, len(data))
	}
}

func testStreamRoundTrip(t *testing.T, factory StoreFactory) {
	t.Helper()
	t.Parallel()
	store := factory(t)
	ctx := context.Background()

	payload := bytes.Repeat([]byte("stream-"), 100)
	if _, err := store.PutStream(ctx, "ns", "streamkey", bytes.NewReader(payload), blobcache.PutOptions{}); err != nil {
		t.Fatalf("PutStream: %v", err)
	}

	reader, ok, _, err := store.Open(ctx, "ns", "streamkey")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !ok {
		t.Fatal("Open ok=false after PutStream")
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), payload) {
		t.Errorf("stream round-trip mismatch: got %d bytes, want %d", buf.Len(), len(payload))
	}
}

func testPathTraversalRejected(t *testing.T, factory StoreFactory) {
	t.Helper()
	t.Parallel()
	store := factory(t)
	ctx := context.Background()

	// Both namespace traversal and key traversal must be rejected before
	// the backend touches the filesystem (or remote bucket prefix).
	traversalCases := []struct {
		namespace string
		key       string
	}{
		{"../escape", "ok"},
		{"ok", "../escape"},
		{"../foo/../escape", "key"},
		{"ns", "subdir/../../escape"},
	}
	for _, tc := range traversalCases {
		_, err := store.Put(ctx, tc.namespace, tc.key, []byte("x"), blobcache.PutOptions{})
		if err == nil {
			t.Errorf("Put(ns=%q, key=%q) succeeded — must reject traversal", tc.namespace, tc.key)
		}
	}
}

func testDeleteNamespace(t *testing.T, factory StoreFactory) {
	t.Helper()
	t.Parallel()
	store := factory(t)
	ctx := context.Background()

	for _, key := range []string{"a", "b", "c"} {
		if _, err := store.Put(ctx, "to-delete", key, []byte("x"), blobcache.PutOptions{}); err != nil {
			t.Fatalf("Put %s: %v", key, err)
		}
	}
	if _, err := store.Put(ctx, "keep", "kept", []byte("k"), blobcache.PutOptions{}); err != nil {
		t.Fatalf("Put keep: %v", err)
	}

	if err := store.DeleteNamespace(ctx, "to-delete"); err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}

	objects, err := store.List(ctx, "to-delete")
	if err != nil {
		t.Fatalf("List to-delete: %v", err)
	}
	if len(objects) != 0 {
		t.Errorf("after DeleteNamespace, List = %v, want empty", objects)
	}

	// Other namespaces stay intact.
	objects, err = store.List(ctx, "keep")
	if err != nil {
		t.Fatalf("List keep: %v", err)
	}
	if len(objects) != 1 || objects[0].Key != "kept" {
		t.Errorf("DeleteNamespace bled across namespaces: List(keep) = %v", objects)
	}
}
