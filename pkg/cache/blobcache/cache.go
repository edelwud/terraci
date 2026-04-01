package blobcache

import (
	"context"
	"time"

	"github.com/edelwud/terraci/pkg/plugin"
)

// Cache provides cache-oriented operations over a blob store while leaving
// higher-level domain serialization and refresh policy to consumers.
type Cache struct {
	store  scopedStore
	policy Policy
}

// New constructs a cache view over a blob store.
func New(store plugin.BlobStore, namespace string, ttl time.Duration) *Cache {
	return NewWithPolicy(store, namespace, Policy{TTL: ttl})
}

// NewWithPolicy constructs a cache view with an explicit cache policy.
func NewWithPolicy(store plugin.BlobStore, namespace string, policy Policy) *Cache {
	return &Cache{
		store:  newBlobStoreScope(store, namespace),
		policy: policy.normalized(),
	}
}

// Dir returns the resolved blob store root directory when exposed by the backend.
func (c *Cache) Dir() string {
	if c.store == nil {
		return ""
	}
	return c.store.Dir()
}

// TTL returns the cache time-to-live duration.
func (c *Cache) TTL() time.Duration {
	return c.policy.TTL
}

// Get reads one cache entry.
func (c *Cache) Get(ctx context.Context, key string) (data []byte, meta plugin.BlobMeta, ok bool, err error) {
	if c.store == nil {
		return nil, plugin.BlobMeta{}, false, ErrStoreNotConfigured
	}
	return c.store.Get(ctx, key)
}

// Put stores one cache entry.
func (c *Cache) Put(ctx context.Context, key string, value []byte, opts PutOptions) (plugin.BlobMeta, error) {
	if c.store == nil {
		return plugin.BlobMeta{}, ErrStoreNotConfigured
	}
	return c.store.Put(ctx, key, value, opts)
}

// Delete removes one cache entry.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if c.store == nil {
		return nil
	}
	return c.store.Delete(ctx, key)
}

// DeleteNamespace clears the entire cache namespace.
func (c *Cache) DeleteNamespace(ctx context.Context) error {
	if c.store == nil {
		return nil
	}
	return c.store.DeleteNamespace(ctx)
}

// List returns cache objects with policy-derived timing info.
func (c *Cache) List(ctx context.Context) ([]Object, error) {
	if c.store == nil {
		return nil, nil
	}

	objects, err := c.store.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]Object, 0, len(objects))
	for _, object := range objects {
		out = append(out, Object{
			Key:       object.Key,
			Meta:      cloneBlobMeta(object.Meta),
			Age:       c.policy.age(object.Meta),
			ExpiresIn: c.policy.expiresIn(object.Meta),
		})
	}
	return out, nil
}

// CleanExpired removes all expired cache entries according to the configured policy.
func (c *Cache) CleanExpired(ctx context.Context) error {
	if c.store == nil {
		return nil
	}

	objects, err := c.store.List(ctx)
	if err != nil {
		return err
	}

	for _, object := range objects {
		if !c.policy.isExpired(object.Meta) {
			continue
		}
		if err := c.store.Delete(ctx, object.Key); err != nil {
			return err
		}
	}

	return nil
}
