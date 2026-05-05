package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

type cachedClient struct {
	base      Client
	cache     plugin.KVCache
	namespace string
	ttl       time.Duration
}

func NewCachedClient(base Client, cache plugin.KVCache, namespace string, ttl time.Duration) Client {
	if base == nil || cache == nil {
		return nil
	}

	return &cachedClient{
		base:      base,
		cache:     cache,
		namespace: namespace,
		ttl:       ttl,
	}
}

func (c *cachedClient) ModuleVersions(
	ctx context.Context,
	address sourceaddr.ModuleAddress,
) ([]string, error) {
	key := cacheKeyForModule(address)

	if versions, ok := c.load(ctx, key); ok {
		return cloneCachedVersions(versions), nil
	}

	versions, err := c.base.ModuleVersions(ctx, address)
	if err != nil {
		return nil, err
	}

	c.store(ctx, key, versions)
	return cloneCachedVersions(versions), nil
}

func (c *cachedClient) ModuleProviderDeps(
	ctx context.Context,
	address sourceaddr.ModuleAddress,
	version string,
) ([]registrymeta.ModuleProviderDep, error) {
	key := fmt.Sprintf("module-deps:%s@%s", addressKey(address), version)

	if payload, ok, err := c.cache.Get(ctx, c.namespace, key); err == nil && ok {
		var deps []registrymeta.ModuleProviderDep
		if err := json.Unmarshal(payload, &deps); err == nil {
			return deps, nil
		}
	}

	deps, err := c.base.ModuleProviderDeps(ctx, address, version)
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

func (c *cachedClient) ProviderVersions(
	ctx context.Context,
	address sourceaddr.ProviderAddress,
) ([]string, error) {
	key := cacheKeyForProvider(address)

	if versions, ok := c.load(ctx, key); ok {
		return cloneCachedVersions(versions), nil
	}

	versions, err := c.base.ProviderVersions(ctx, address)
	if err != nil {
		return nil, err
	}

	c.store(ctx, key, versions)
	return cloneCachedVersions(versions), nil
}

func (c *cachedClient) ProviderPlatforms(
	ctx context.Context,
	address sourceaddr.ProviderAddress,
	version string,
) ([]string, error) {
	key := cacheKeyForProviderPlatforms(address, version)

	if platforms, ok := c.loadPlatforms(ctx, key); ok {
		return cloneCachedPlatforms(platforms), nil
	}

	platforms, err := c.base.ProviderPlatforms(ctx, address, version)
	if err != nil {
		return nil, err
	}

	c.storePlatforms(ctx, key, platforms)
	return cloneCachedPlatforms(platforms), nil
}

func (c *cachedClient) ProviderPackage(
	ctx context.Context,
	address sourceaddr.ProviderAddress,
	version, platform string,
) (*registrymeta.ProviderPackage, error) {
	return c.base.ProviderPackage(ctx, address, version, platform)
}

func (c *cachedClient) load(ctx context.Context, key string) ([]string, bool) {
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

func (c *cachedClient) loadPlatforms(ctx context.Context, key string) ([]string, bool) {
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

func (c *cachedClient) store(ctx context.Context, key string, versions []string) {
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

func (c *cachedClient) storePlatforms(ctx context.Context, key string, platforms []string) {
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

func cacheKeyForModule(address sourceaddr.ModuleAddress) string {
	return "module:" + addressKey(address)
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

func cacheKeyForProvider(address sourceaddr.ProviderAddress) string {
	return "provider:" + providerKey(address)
}

func cacheKeyForProviderPlatforms(address sourceaddr.ProviderAddress, version string) string {
	return fmt.Sprintf("provider-platforms:%s@%s", providerKey(address), version)
}

func addressKey(address sourceaddr.ModuleAddress) string {
	return fmt.Sprintf("%s/%s/%s/%s", address.Hostname, address.Namespace, address.Name, address.Provider)
}

func providerKey(address sourceaddr.ProviderAddress) string {
	return fmt.Sprintf("%s/%s/%s", address.Hostname, address.Namespace, address.Type)
}
