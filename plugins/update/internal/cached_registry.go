package updateengine

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
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
	namespace, name, provider string,
) ([]string, error) {
	key := cacheKeyForModule(namespace, name, provider)

	if versions, ok := c.load(ctx, key); ok {
		return cloneCachedVersions(versions), nil
	}

	versions, err := c.base.ModuleVersions(ctx, namespace, name, provider)
	if err != nil {
		return nil, err
	}

	c.store(ctx, key, versions)
	return cloneCachedVersions(versions), nil
}

func (c *cachedRegistryClient) ProviderVersions(
	ctx context.Context,
	namespace, typeName string,
) ([]string, error) {
	key := cacheKeyForProvider(namespace, typeName)

	if versions, ok := c.load(ctx, key); ok {
		return cloneCachedVersions(versions), nil
	}

	versions, err := c.base.ProviderVersions(ctx, namespace, typeName)
	if err != nil {
		return nil, err
	}

	c.store(ctx, key, versions)
	return cloneCachedVersions(versions), nil
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

func cacheKeyForModule(namespace, name, provider string) string {
	return fmt.Sprintf("module:%s/%s/%s", namespace, name, provider)
}

func cloneCachedVersions(versions []string) []string {
	if len(versions) == 0 {
		return nil
	}

	return slices.Clone(versions)
}

func cacheKeyForProvider(namespace, typeName string) string {
	return fmt.Sprintf("provider:%s/%s", namespace, typeName)
}
