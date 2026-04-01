package plugin

import (
	"context"
	"time"
)

// KVCache is a pluggable key/value cache backend.
//
// Values are stored as opaque bytes. Consumers own serialization, key layout,
// namespaces, and write-time TTL selection.
type KVCache interface {
	Get(ctx context.Context, namespace, key string) ([]byte, bool, error)
	Set(ctx context.Context, namespace, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, namespace, key string) error
	DeleteNamespace(ctx context.Context, namespace string) error
}

// KVCacheProvider creates a KV cache backend for plugin consumers.
//
// Providers are registered like any other TerraCi plugin and resolved by name
// through the global plugin registry.
type KVCacheProvider interface {
	Plugin
	NewKVCache(ctx context.Context, appCtx *AppContext) (KVCache, error)
}
