package tfupdateengine

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
)

type cachedRegistryClient struct {
	base      RegistryClient
	cache     plugin.KVCache
	namespace string
	ttl       time.Duration
}

func NewCachedRegistryClient(base RegistryClient, cache plugin.KVCache, namespace string, ttl time.Duration) RegistryClient {
	if base == nil || cache == nil {
		return nil
	}

	return &cachedRegistryClient{
		base:      base,
		cache:     cache,
		namespace: namespace,
		ttl:       ttl,
	}
}

func (c *cachedRegistryClient) ModuleVersions(
	ctx context.Context,
	hostname, namespace, name, provider string,
) ([]string, error) {
	key := cacheKeyForModule(hostname, namespace, name, provider)

	if versions, ok := c.load(ctx, key); ok {
		return cloneCachedVersions(versions), nil
	}

	versions, err := c.base.ModuleVersions(ctx, hostname, namespace, name, provider)
	if err != nil {
		return nil, err
	}

	c.store(ctx, key, versions)
	return cloneCachedVersions(versions), nil
}

func (c *cachedRegistryClient) ModuleProviderDeps(
	ctx context.Context,
	hostname, namespace, name, provider, version string,
) ([]registrymeta.ModuleProviderDep, error) {
	key := fmt.Sprintf("module-deps:%s/%s/%s/%s@%s", hostname, namespace, name, provider, version)

	if payload, ok, err := c.cache.Get(ctx, c.namespace, key); err == nil && ok {
		var deps []registrymeta.ModuleProviderDep
		if err := json.Unmarshal(payload, &deps); err == nil {
			return deps, nil
		}
	}

	deps, err := c.base.ModuleProviderDeps(ctx, hostname, namespace, name, provider, version)
	if err != nil {
		return nil, err
	}

	if payload, err := json.Marshal(deps); err == nil {
		if setErr := c.cache.Set(ctx, c.namespace, key, payload, c.ttl); setErr != nil {
			log.WithError(setErr).
				WithField("backend_namespace", c.namespace).
				WithField("cache_key", key).
				Debug("update: cache write failed; continuing without cached result")
		}
	}

	return deps, nil
}

func (c *cachedRegistryClient) ProviderVersions(
	ctx context.Context,
	hostname, namespace, typeName string,
) ([]string, error) {
	key := cacheKeyForProvider(hostname, namespace, typeName)

	if versions, ok := c.load(ctx, key); ok {
		return cloneCachedVersions(versions), nil
	}

	versions, err := c.base.ProviderVersions(ctx, hostname, namespace, typeName)
	if err != nil {
		return nil, err
	}

	c.store(ctx, key, versions)
	return cloneCachedVersions(versions), nil
}

func (c *cachedRegistryClient) ProviderPlatforms(
	ctx context.Context,
	hostname, namespace, typeName, version string,
) ([]string, error) {
	key := cacheKeyForProviderPlatforms(hostname, namespace, typeName, version)

	if platforms, ok := c.loadPlatforms(ctx, key); ok {
		return cloneCachedPlatforms(platforms), nil
	}

	platforms, err := c.base.ProviderPlatforms(ctx, hostname, namespace, typeName, version)
	if err != nil {
		return nil, err
	}

	c.storePlatforms(ctx, key, platforms)
	return cloneCachedPlatforms(platforms), nil
}

func (c *cachedRegistryClient) ProviderPackage(
	ctx context.Context,
	hostname, namespace, typeName, version, platform string,
) (*registrymeta.ProviderPackage, error) {
	return c.base.ProviderPackage(ctx, hostname, namespace, typeName, version, platform)
}

func (c *cachedRegistryClient) load(ctx context.Context, key string) ([]string, bool) {
	payload, ok, err := c.cache.Get(ctx, c.namespace, key)
	if err != nil {
		log.WithError(err).
			WithField("backend_namespace", c.namespace).
			WithField("cache_key", key).
			Debug("update: cache read failed; falling back to registry")
		return nil, false
	}
	if !ok {
		return nil, false
	}

	var versions []string
	if err := json.Unmarshal(payload, &versions); err != nil {
		log.WithError(err).
			WithField("backend_namespace", c.namespace).
			WithField("cache_key", key).
			Debug("update: cache payload decode failed; refreshing from registry")
		return nil, false
	}

	return versions, true
}

func (c *cachedRegistryClient) loadPlatforms(ctx context.Context, key string) ([]string, bool) {
	payload, ok, err := c.cache.Get(ctx, c.namespace, key)
	if err != nil {
		log.WithError(err).
			WithField("backend_namespace", c.namespace).
			WithField("cache_key", key).
			Debug("update: cache read failed; falling back to registry")
		return nil, false
	}
	if !ok {
		return nil, false
	}

	var platforms []string
	if err := json.Unmarshal(payload, &platforms); err != nil {
		log.WithError(err).
			WithField("backend_namespace", c.namespace).
			WithField("cache_key", key).
			Debug("update: cache payload decode failed; refreshing from registry")
		return nil, false
	}

	return platforms, true
}

func (c *cachedRegistryClient) store(ctx context.Context, key string, versions []string) {
	payload, err := json.Marshal(cloneCachedVersions(versions))
	if err != nil {
		log.WithError(err).
			WithField("backend_namespace", c.namespace).
			WithField("cache_key", key).
			Debug("update: cache payload encode failed; skipping cache write")
		return
	}

	if err := c.cache.Set(ctx, c.namespace, key, payload, c.ttl); err != nil {
		log.WithError(err).
			WithField("backend_namespace", c.namespace).
			WithField("cache_key", key).
			Debug("update: cache write failed; continuing without cached result")
	}
}

func (c *cachedRegistryClient) storePlatforms(ctx context.Context, key string, platforms []string) {
	payload, err := json.Marshal(cloneCachedPlatforms(platforms))
	if err != nil {
		log.WithError(err).
			WithField("backend_namespace", c.namespace).
			WithField("cache_key", key).
			Debug("update: cache payload encode failed; skipping cache write")
		return
	}

	if err := c.cache.Set(ctx, c.namespace, key, payload, c.ttl); err != nil {
		log.WithError(err).
			WithField("backend_namespace", c.namespace).
			WithField("cache_key", key).
			Debug("update: cache write failed; continuing without cached result")
	}
}

func cacheKeyForModule(hostname, namespace, name, provider string) string {
	return fmt.Sprintf("module:%s/%s/%s/%s", hostname, namespace, name, provider)
}

func cloneCachedVersions(versions []string) []string {
	if len(versions) == 0 {
		return nil
	}

	return slices.Clone(versions)
}

func cloneCachedPlatforms(platforms []string) []string {
	if len(platforms) == 0 {
		return nil
	}

	return slices.Clone(platforms)
}

func cacheKeyForProvider(hostname, namespace, typeName string) string {
	return fmt.Sprintf("provider:%s/%s/%s", hostname, namespace, typeName)
}

func cacheKeyForProviderPlatforms(hostname, namespace, typeName, version string) string {
	return fmt.Sprintf("provider-platforms:%s/%s/%s@%s", hostname, namespace, typeName, version)
}
