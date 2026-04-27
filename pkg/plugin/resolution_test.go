package plugin

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
)

type resolutionTestResolver struct {
	plugin       Plugin
	blobProvider BlobStoreProvider
	kvProvider   KVCacheProvider
	contribs     []*pipeline.Contribution
}

func (r resolutionTestResolver) GetPlugin(string) (Plugin, bool) {
	return r.plugin, r.plugin != nil
}

func (r resolutionTestResolver) ResolveBlobStoreProvider(string) (BlobStoreProvider, error) {
	return r.blobProvider, nil
}

func (r resolutionTestResolver) ResolveKVCacheProvider(string) (KVCacheProvider, error) {
	return r.kvProvider, nil
}

func (r resolutionTestResolver) CollectContributions(*AppContext) []*pipeline.Contribution {
	return r.contribs
}

type resolutionTestBlobProvider struct {
	contextTestPlugin
}

func (resolutionTestBlobProvider) NewBlobStore(context.Context, *AppContext) (BlobStore, error) {
	return nil, nil
}

type resolutionTestKVProvider struct {
	contextTestPlugin
}

func (resolutionTestKVProvider) NewKVCache(context.Context, *AppContext) (KVCache, error) {
	return nil, nil
}

func TestResolutionHelpersDelegateToContextResolver(t *testing.T) {
	blob := &resolutionTestBlobProvider{contextTestPlugin{name: "blob"}}
	kv := &resolutionTestKVProvider{contextTestPlugin{name: "kv"}}
	contribs := []*pipeline.Contribution{{Jobs: []pipeline.ContributedJob{{Name: "job"}}}}
	ctx := NewAppContext(nil, "/tmp", "/tmp/.terraci", "test", nil, resolutionTestResolver{
		blobProvider: blob,
		kvProvider:   kv,
		contribs:     contribs,
	})

	gotBlob, err := ResolveBlobStoreProvider(ctx, "blob")
	if err != nil {
		t.Fatalf("ResolveBlobStoreProvider() error = %v", err)
	}
	if gotBlob.Name() != "blob" {
		t.Fatalf("ResolveBlobStoreProvider() = %q, want blob", gotBlob.Name())
	}

	gotKV, err := ResolveKVCacheProvider(ctx, "kv")
	if err != nil {
		t.Fatalf("ResolveKVCacheProvider() error = %v", err)
	}
	if gotKV.Name() != "kv" {
		t.Fatalf("ResolveKVCacheProvider() = %q, want kv", gotKV.Name())
	}

	if got := CollectContributions(ctx); len(got) != 1 || got[0].Jobs[0].Name != "job" {
		t.Fatalf("CollectContributions() = %#v, want job contribution", got)
	}
}

func TestResolutionHelpersRejectUnsupportedResolver(t *testing.T) {
	ctx := NewAppContext(nil, "/tmp", "/tmp/.terraci", "test", nil, contextTestResolver{})

	if _, err := ResolveCIProvider(ctx); err == nil {
		t.Fatal("ResolveCIProvider() error = nil, want unsupported resolver error")
	}
	if _, err := ResolveChangeDetector(ctx); err == nil {
		t.Fatal("ResolveChangeDetector() error = nil, want unsupported resolver error")
	}
	if _, err := ResolveKVCacheProvider(ctx, "kv"); err == nil {
		t.Fatal("ResolveKVCacheProvider() error = nil, want unsupported resolver error")
	}
	if _, err := ResolveBlobStoreProvider(ctx, "blob"); err == nil {
		t.Fatal("ResolveBlobStoreProvider() error = nil, want unsupported resolver error")
	}
	if got := CollectContributions(ctx); got != nil {
		t.Fatalf("CollectContributions() = %#v, want nil", got)
	}
}
