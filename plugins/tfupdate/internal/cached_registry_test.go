package tfupdateengine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
)

type countingRegistryClient struct {
	moduleVersions        []string
	providerVersions      []string
	providerPlatforms     []string
	providerPackage       *registrymeta.ProviderPackage
	moduleErr             error
	providerErr           error
	moduleCalls           int
	providerCalls         int
	providerPlatformCalls int
	providerPackageCalls  int
}

func (c *countingRegistryClient) ModuleVersions(_ context.Context, _, _, _, _ string) ([]string, error) {
	c.moduleCalls++
	return c.moduleVersions, c.moduleErr
}

func (c *countingRegistryClient) ModuleProviderDeps(_ context.Context, _, _, _, _, _ string) ([]registrymeta.ModuleProviderDep, error) {
	return nil, nil
}

func (c *countingRegistryClient) ProviderVersions(_ context.Context, _, _, _ string) ([]string, error) {
	c.providerCalls++
	return c.providerVersions, c.providerErr
}

func (c *countingRegistryClient) ProviderPlatforms(_ context.Context, _, _, _, _ string) ([]string, error) {
	c.providerPlatformCalls++
	return cloneCachedPlatforms(c.providerPlatforms), c.providerErr
}

func (c *countingRegistryClient) ProviderPackage(_ context.Context, _, _, _, _, _ string) (*registrymeta.ProviderPackage, error) {
	c.providerPackageCalls++
	if c.providerPackage == nil {
		return nil, c.providerErr
	}
	copyPkg := *c.providerPackage
	return &copyPkg, c.providerErr
}

type memoryKVCache struct {
	entries map[string][]byte
}

func newMemoryKVCache() *memoryKVCache {
	return &memoryKVCache{entries: make(map[string][]byte)}
}

func (c *memoryKVCache) Get(_ context.Context, namespace, key string) (value []byte, found bool, err error) {
	value, ok := c.entries[namespace+"|"+key]
	return append([]byte(nil), value...), ok, nil
}

func (c *memoryKVCache) Set(_ context.Context, namespace, key string, value []byte, _ time.Duration) error {
	c.entries[namespace+"|"+key] = append([]byte(nil), value...)
	return nil
}

func (c *memoryKVCache) Delete(_ context.Context, namespace, key string) error {
	delete(c.entries, namespace+"|"+key)
	return nil
}

func (c *memoryKVCache) DeleteNamespace(_ context.Context, namespace string) error {
	for key := range c.entries {
		if len(key) >= len(namespace)+1 && key[:len(namespace)+1] == namespace+"|" {
			delete(c.entries, key)
		}
	}
	return nil
}

type failingKVCache struct {
	getErr error
	setErr error
}

func (c failingKVCache) Get(_ context.Context, _, _ string) (value []byte, found bool, err error) {
	return nil, false, c.getErr
}

func (c failingKVCache) Set(_ context.Context, _, _ string, _ []byte, _ time.Duration) error {
	return c.setErr
}

func (c failingKVCache) Delete(_ context.Context, _, _ string) error { return nil }
func (c failingKVCache) DeleteNamespace(_ context.Context, _ string) error {
	return nil
}

func TestCachedRegistryClient_ModuleVersions(t *testing.T) {
	base := &countingRegistryClient{moduleVersions: []string{"1.0.0", "1.1.0"}}
	client := NewCachedRegistryClient(base, newMemoryKVCache(), DefaultCacheNamespace, time.Hour)

	first, err := client.ModuleVersions(context.Background(), "registry.terraform.io", "hashicorp", "consul", "aws")
	if err != nil {
		t.Fatalf("ModuleVersions() first error = %v", err)
	}
	second, err := client.ModuleVersions(context.Background(), "registry.terraform.io", "hashicorp", "consul", "aws")
	if err != nil {
		t.Fatalf("ModuleVersions() second error = %v", err)
	}

	if base.moduleCalls != 1 {
		t.Fatalf("moduleCalls = %d, want 1", base.moduleCalls)
	}
	first[0] = "mutated"
	if second[0] != "1.0.0" {
		t.Fatalf("cached versions were mutated through caller slice: got %q", second[0])
	}
}

func TestCachedRegistryClient_ProviderVersions(t *testing.T) {
	base := &countingRegistryClient{providerVersions: []string{"5.0.0", "5.1.0"}}
	client := NewCachedRegistryClient(base, newMemoryKVCache(), DefaultCacheNamespace, time.Hour)

	_, err := client.ProviderVersions(context.Background(), "registry.terraform.io", "hashicorp", "aws")
	if err != nil {
		t.Fatalf("ProviderVersions() first error = %v", err)
	}
	_, err = client.ProviderVersions(context.Background(), "registry.terraform.io", "hashicorp", "aws")
	if err != nil {
		t.Fatalf("ProviderVersions() second error = %v", err)
	}

	if base.providerCalls != 1 {
		t.Fatalf("providerCalls = %d, want 1", base.providerCalls)
	}
}

func TestCachedRegistryClient_ProviderVersions_CacheKeyIncludesHostname(t *testing.T) {
	base := &countingRegistryClient{providerVersions: []string{"5.0.0", "5.1.0"}}
	client := NewCachedRegistryClient(base, newMemoryKVCache(), DefaultCacheNamespace, time.Hour)

	_, err := client.ProviderVersions(context.Background(), "registry.terraform.io", "hashicorp", "aws")
	if err != nil {
		t.Fatalf("ProviderVersions() terraform error = %v", err)
	}
	_, err = client.ProviderVersions(context.Background(), "registry.opentofu.org", "hashicorp", "aws")
	if err != nil {
		t.Fatalf("ProviderVersions() opentofu error = %v", err)
	}

	if base.providerCalls != 2 {
		t.Fatalf("providerCalls = %d, want 2", base.providerCalls)
	}
}

func TestCachedRegistryClient_ProviderPlatforms(t *testing.T) {
	base := &countingRegistryClient{
		providerPlatforms: []string{"darwin_arm64", "linux_amd64"},
	}
	client := NewCachedRegistryClient(base, newMemoryKVCache(), DefaultCacheNamespace, time.Hour)

	first, err := client.ProviderPlatforms(context.Background(), "registry.terraform.io", "hashicorp", "aws", "5.0.0")
	if err != nil {
		t.Fatalf("ProviderPlatforms() first error = %v", err)
	}
	second, err := client.ProviderPlatforms(context.Background(), "registry.terraform.io", "hashicorp", "aws", "5.0.0")
	if err != nil {
		t.Fatalf("ProviderPlatforms() second error = %v", err)
	}

	if base.providerPlatformCalls != 1 {
		t.Fatalf("providerPlatformCalls = %d, want 1", base.providerPlatformCalls)
	}
	first[0] = "mutated"
	if second[0] != "darwin_arm64" {
		t.Fatalf("cached platforms were mutated through caller slice: got %q", second[0])
	}
}

func TestCachedRegistryClient_DecodeFailureRefreshesFromRegistry(t *testing.T) {
	cache := newMemoryKVCache()
	key := cacheKeyForModule("registry.terraform.io", "hashicorp", "consul", "aws")
	cache.entries[DefaultCacheNamespace+"|"+key] = []byte("not-json")

	base := &countingRegistryClient{moduleVersions: []string{"1.2.3"}}
	client := NewCachedRegistryClient(base, cache, DefaultCacheNamespace, time.Hour)

	got, err := client.ModuleVersions(context.Background(), "registry.terraform.io", "hashicorp", "consul", "aws")
	if err != nil {
		t.Fatalf("ModuleVersions() error = %v", err)
	}
	if base.moduleCalls != 1 {
		t.Fatalf("moduleCalls = %d, want 1", base.moduleCalls)
	}
	if len(got) != 1 || got[0] != "1.2.3" {
		t.Fatalf("ModuleVersions() = %v, want [1.2.3]", got)
	}
}

func TestCachedRegistryClient_CacheFailureFallsBackToRegistry(t *testing.T) {
	base := &countingRegistryClient{providerVersions: []string{"6.0.0"}}
	client := NewCachedRegistryClient(base, failingKVCache{getErr: errors.New("boom")}, DefaultCacheNamespace, time.Hour)

	got, err := client.ProviderVersions(context.Background(), "registry.terraform.io", "hashicorp", "aws")
	if err != nil {
		t.Fatalf("ProviderVersions() error = %v", err)
	}
	if base.providerCalls != 1 {
		t.Fatalf("providerCalls = %d, want 1", base.providerCalls)
	}
	if len(got) != 1 || got[0] != "6.0.0" {
		t.Fatalf("ProviderVersions() = %v, want [6.0.0]", got)
	}
}

func TestNewCachedRegistryClient_NilDeps(t *testing.T) {
	if got := NewCachedRegistryClient(nil, newMemoryKVCache(), DefaultCacheNamespace, time.Hour); got != nil {
		t.Fatal("expected nil client when base registry is nil")
	}
	if got := NewCachedRegistryClient(&countingRegistryClient{}, nil, DefaultCacheNamespace, time.Hour); got != nil {
		t.Fatal("expected nil client when cache backend is nil")
	}
}

var _ plugin.KVCache = (*memoryKVCache)(nil)
