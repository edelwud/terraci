package blobcache

import (
	"context"
	"errors"
	"io"
	"testing"
)

type stubBlobStore struct{}

func (stubBlobStore) Get(context.Context, string, string) (data []byte, ok bool, meta Meta, err error) {
	return nil, false, Meta{}, nil
}
func (stubBlobStore) Put(context.Context, string, string, []byte, PutOptions) (Meta, error) {
	return Meta{}, nil
}
func (stubBlobStore) Open(context.Context, string, string) (io.ReadCloser, bool, Meta, error) {
	return nil, false, Meta{}, nil
}
func (stubBlobStore) PutStream(context.Context, string, string, io.Reader, PutOptions) (Meta, error) {
	return Meta{}, nil
}
func (stubBlobStore) Delete(context.Context, string, string) error  { return nil }
func (stubBlobStore) DeleteNamespace(context.Context, string) error { return nil }
func (stubBlobStore) List(context.Context, string) ([]Object, error) {
	return nil, nil
}

type blobStoreWithInspector struct {
	stubBlobStore
	root string
}

func (s blobStoreWithInspector) BlobStoreRootDir() string {
	return s.root
}

type blobStoreWithDescription struct {
	stubBlobStore
	info Info
	err  error
}

func (s blobStoreWithDescription) DescribeBlobStore() Info {
	return s.info
}

func (s blobStoreWithDescription) CheckBlobStore(context.Context) error {
	return s.err
}

func TestDescribe_UsesDescriber(t *testing.T) {
	info := Describe(blobStoreWithDescription{
		info: Info{
			Backend: "diskblob",
			Root:    "/tmp/cache",
		},
	}, "fallback")

	if info.Backend != "diskblob" || info.Root != "/tmp/cache" {
		t.Fatalf("Describe() = %+v, want describer info", info)
	}
}

func TestDescribe_FallsBackToInspector(t *testing.T) {
	info := Describe(blobStoreWithInspector{root: "/tmp/legacy"}, "fallback")
	if info.Backend != "fallback" {
		t.Fatalf("Describe().Backend = %q, want fallback", info.Backend)
	}
	if info.Root != "/tmp/legacy" {
		t.Fatalf("Describe().Root = %q, want /tmp/legacy", info.Root)
	}
}

func TestDescribe_FallsBackToBackendName(t *testing.T) {
	info := Describe(stubBlobStore{}, "fallback")
	if info.Backend != "fallback" || info.Root != "" {
		t.Fatalf("Describe() = %+v, want fallback-only info", info)
	}
}

func TestCheck_Optional(t *testing.T) {
	if err := Check(context.Background(), stubBlobStore{}); err != nil {
		t.Fatalf("Check() error = %v, want nil", err)
	}

	wantErr := errors.New("boom")
	err := Check(context.Background(), blobStoreWithDescription{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Check() error = %v, want %v", err, wantErr)
	}
}
