package plugin

import (
	"errors"

	"github.com/edelwud/terraci/pkg/pipeline"
)

type ciProviderResolver interface {
	ResolveCIProvider() (*ResolvedCIProvider, error)
}

type changeDetectorResolver interface {
	ResolveChangeDetector() (ChangeDetectionProvider, error)
}

type kvCacheResolver interface {
	ResolveKVCacheProvider(name string) (KVCacheProvider, error)
}

type blobStoreResolver interface {
	ResolveBlobStoreProvider(name string) (BlobStoreProvider, error)
}

type contributionCollector interface {
	CollectContributions(ctx *AppContext) []*pipeline.Contribution
}

// ResolveCIProvider resolves the active CI provider through the context-bound
// plugin registry.
func ResolveCIProvider(ctx *AppContext) (*ResolvedCIProvider, error) {
	if ctx == nil {
		return nil, errors.New("plugin context is nil")
	}
	resolver, ok := ctx.resolver.(ciProviderResolver)
	if !ok {
		return nil, errors.New("plugin resolver does not support CI provider resolution")
	}
	return resolver.ResolveCIProvider()
}

// ResolveChangeDetector resolves the active change detector through the
// context-bound plugin registry.
func ResolveChangeDetector(ctx *AppContext) (ChangeDetectionProvider, error) {
	if ctx == nil {
		return nil, errors.New("plugin context is nil")
	}
	resolver, ok := ctx.resolver.(changeDetectorResolver)
	if !ok {
		return nil, errors.New("plugin resolver does not support change detector resolution")
	}
	return resolver.ResolveChangeDetector()
}

// ResolveKVCacheProvider resolves a named KV cache backend through the
// context-bound plugin registry.
func ResolveKVCacheProvider(ctx *AppContext, name string) (KVCacheProvider, error) {
	if ctx == nil {
		return nil, errors.New("plugin context is nil")
	}
	resolver, ok := ctx.resolver.(kvCacheResolver)
	if !ok {
		return nil, errors.New("plugin resolver does not support KV cache backend resolution")
	}
	return resolver.ResolveKVCacheProvider(name)
}

// ResolveBlobStoreProvider resolves a named blob backend through the
// context-bound plugin registry.
func ResolveBlobStoreProvider(ctx *AppContext, name string) (BlobStoreProvider, error) {
	if ctx == nil {
		return nil, errors.New("plugin context is nil")
	}
	resolver, ok := ctx.resolver.(blobStoreResolver)
	if !ok {
		return nil, errors.New("plugin resolver does not support blob backend resolution")
	}
	return resolver.ResolveBlobStoreProvider(name)
}

// CollectContributions gathers pipeline contributions through the
// context-bound plugin registry.
func CollectContributions(ctx *AppContext) []*pipeline.Contribution {
	if ctx == nil {
		return nil
	}
	collector, ok := ctx.resolver.(contributionCollector)
	if !ok {
		return nil
	}
	return collector.CollectContributions(ctx)
}
