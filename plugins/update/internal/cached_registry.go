package updateengine

import (
	"context"
	"slices"
	"sync"
)

type cachedRegistryClient struct {
	base RegistryClient

	mu        sync.Mutex
	modules   map[string]cachedVersionResult
	providers map[string]cachedVersionResult
}

type cachedVersionResult struct {
	versions []string
	err      error
}

func NewCachedRegistryClient(base RegistryClient) RegistryClient {
	if base == nil {
		return nil
	}

	return &cachedRegistryClient{
		base:      base,
		modules:   make(map[string]cachedVersionResult),
		providers: make(map[string]cachedVersionResult),
	}
}

func (c *cachedRegistryClient) ModuleVersions(
	ctx context.Context,
	namespace, name, provider string,
) ([]string, error) {
	key := namespace + "/" + name + "/" + provider

	if result, ok := c.lookupModule(key); ok {
		return cloneCachedVersions(result.versions), result.err
	}

	versions, err := c.base.ModuleVersions(ctx, namespace, name, provider)
	result := cachedVersionResult{
		versions: cloneCachedVersions(versions),
		err:      err,
	}

	c.storeModule(key, result)
	return cloneCachedVersions(result.versions), result.err
}

func (c *cachedRegistryClient) ProviderVersions(
	ctx context.Context,
	namespace, typeName string,
) ([]string, error) {
	key := namespace + "/" + typeName

	if result, ok := c.lookupProvider(key); ok {
		return cloneCachedVersions(result.versions), result.err
	}

	versions, err := c.base.ProviderVersions(ctx, namespace, typeName)
	result := cachedVersionResult{
		versions: cloneCachedVersions(versions),
		err:      err,
	}

	c.storeProvider(key, result)
	return cloneCachedVersions(result.versions), result.err
}

func (c *cachedRegistryClient) lookupModule(key string) (cachedVersionResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result, ok := c.modules[key]
	return result, ok
}

func (c *cachedRegistryClient) storeModule(key string, result cachedVersionResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.modules[key] = result
}

func (c *cachedRegistryClient) lookupProvider(key string) (cachedVersionResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result, ok := c.providers[key]
	return result, ok
}

func (c *cachedRegistryClient) storeProvider(key string, result cachedVersionResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.providers[key] = result
}

func cloneCachedVersions(versions []string) []string {
	if len(versions) == 0 {
		return nil
	}

	return slices.Clone(versions)
}
