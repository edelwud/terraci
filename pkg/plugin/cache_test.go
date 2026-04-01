package plugin

import (
	"context"
	"errors"
	"io"
	"testing"
)

type stubBlobStore struct{}

func (stubBlobStore) Get(context.Context, string, string) (data []byte, ok bool, meta BlobMeta, err error) {
	return nil, false, BlobMeta{}, nil
}
func (stubBlobStore) Put(context.Context, string, string, []byte, PutBlobOptions) (BlobMeta, error) {
	return BlobMeta{}, nil
}
func (stubBlobStore) Open(context.Context, string, string) (io.ReadCloser, bool, BlobMeta, error) {
	return nil, false, BlobMeta{}, nil
}
func (stubBlobStore) PutStream(context.Context, string, string, io.Reader, PutBlobOptions) (BlobMeta, error) {
	return BlobMeta{}, nil
}
func (stubBlobStore) Delete(context.Context, string, string) error       { return nil }
func (stubBlobStore) DeleteNamespace(context.Context, string) error      { return nil }
func (stubBlobStore) List(context.Context, string) ([]BlobObject, error) { return nil, nil }

type blobStoreWithInspector struct {
	stubBlobStore
	root string
}

func (s blobStoreWithInspector) BlobStoreRootDir() string {
	return s.root
}

type blobStoreWithDescription struct {
	stubBlobStore
	info BlobStoreInfo
	err  error
}

func (s blobStoreWithDescription) DescribeBlobStore() BlobStoreInfo {
	return s.info
}

func (s blobStoreWithDescription) CheckBlobStore(context.Context) error {
	return s.err
}

func TestDescribeBlobStore_UsesDescriber(t *testing.T) {
	info := DescribeBlobStore(blobStoreWithDescription{
		info: BlobStoreInfo{
			Backend: "diskblob",
			Root:    "/tmp/cache",
		},
	}, "fallback")

	if info.Backend != "diskblob" || info.Root != "/tmp/cache" {
		t.Fatalf("DescribeBlobStore() = %+v, want describer info", info)
	}
}

func TestDescribeBlobStore_FallsBackToInspector(t *testing.T) {
	info := DescribeBlobStore(blobStoreWithInspector{root: "/tmp/legacy"}, "fallback")
	if info.Backend != "fallback" {
		t.Fatalf("DescribeBlobStore().Backend = %q, want fallback", info.Backend)
	}
	if info.Root != "/tmp/legacy" {
		t.Fatalf("DescribeBlobStore().Root = %q, want /tmp/legacy", info.Root)
	}
}

func TestDescribeBlobStore_FallsBackToBackendName(t *testing.T) {
	info := DescribeBlobStore(stubBlobStore{}, "fallback")
	if info.Backend != "fallback" || info.Root != "" {
		t.Fatalf("DescribeBlobStore() = %+v, want fallback-only info", info)
	}
}

func TestCheckBlobStore_Optional(t *testing.T) {
	if err := CheckBlobStore(context.Background(), stubBlobStore{}); err != nil {
		t.Fatalf("CheckBlobStore() error = %v, want nil", err)
	}

	wantErr := errors.New("boom")
	err := CheckBlobStore(context.Background(), blobStoreWithDescription{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("CheckBlobStore() error = %v, want %v", err, wantErr)
	}
}
