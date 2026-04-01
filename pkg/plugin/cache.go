package plugin

import (
	"context"
	"io"
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

// PutBlobOptions controls how a blob is persisted by the backend.
type PutBlobOptions struct {
	ContentType string
	ExpiresAt   *time.Time
	Metadata    map[string]string
}

// BlobMeta describes a stored blob object.
type BlobMeta struct {
	Size        int64
	UpdatedAt   time.Time
	ExpiresAt   *time.Time
	ETag        string
	ContentType string
	Metadata    map[string]string
}

// BlobObject describes a listed blob object.
type BlobObject struct {
	Key  string
	Meta BlobMeta
}

// BlobStore stores opaque binary objects addressed by namespace + key.
//
// Consumers own key layout, serialization, TTL policy, and stale fallback
// semantics. Backends only persist bytes and metadata.
type BlobStore interface {
	Get(ctx context.Context, namespace, key string) ([]byte, bool, BlobMeta, error)
	Put(ctx context.Context, namespace, key string, value []byte, opts PutBlobOptions) (BlobMeta, error)
	Open(ctx context.Context, namespace, key string) (io.ReadCloser, bool, BlobMeta, error)
	PutStream(ctx context.Context, namespace, key string, r io.Reader, opts PutBlobOptions) (BlobMeta, error)
	Delete(ctx context.Context, namespace, key string) error
	DeleteNamespace(ctx context.Context, namespace string) error
	List(ctx context.Context, namespace string) ([]BlobObject, error)
}

// BlobStoreProvider creates a blob store backend for plugin consumers.
type BlobStoreProvider interface {
	Plugin
	NewBlobStore(ctx context.Context, appCtx *AppContext) (BlobStore, error)
}

// BlobStoreOptions carries optional backend-specific initialization overrides.
type BlobStoreOptions struct {
	RootDir string
}

// BlobStoreProviderWithOptions allows a backend provider to accept optional
// initialization overrides from a consumer.
type BlobStoreProviderWithOptions interface {
	BlobStoreProvider
	NewBlobStoreWithOptions(ctx context.Context, appCtx *AppContext, opts BlobStoreOptions) (BlobStore, error)
}

// BlobStoreInspector exposes optional store details for diagnostics.
type BlobStoreInspector interface {
	BlobStoreRootDir() string
}

// BlobStoreInfo describes optional backend diagnostics exposed by a blob store.
type BlobStoreInfo struct {
	Backend                 string
	Root                    string
	SupportsList            bool
	SupportsStream          bool
	SupportsDeleteNamespace bool
}

// BlobStoreDescriber exposes optional diagnostics for a blob store backend.
type BlobStoreDescriber interface {
	DescribeBlobStore() BlobStoreInfo
}

// BlobStoreHealthChecker exposes an optional health check for a blob store backend.
type BlobStoreHealthChecker interface {
	CheckBlobStore(ctx context.Context) error
}

// DescribeBlobStore returns the optional backend diagnostics exposed by a blob
// store, applying a fallback backend name when the store omits one.
func DescribeBlobStore(store BlobStore, fallbackBackend string) BlobStoreInfo {
	if store == nil {
		return BlobStoreInfo{Backend: fallbackBackend}
	}

	if describer, ok := store.(BlobStoreDescriber); ok {
		info := describer.DescribeBlobStore()
		if info.Backend == "" {
			info.Backend = fallbackBackend
		}
		if info.Root != "" || info.Backend != "" || info.SupportsList || info.SupportsStream || info.SupportsDeleteNamespace {
			return info
		}
	}

	info := BlobStoreInfo{Backend: fallbackBackend}
	if inspector, ok := store.(BlobStoreInspector); ok {
		info.Root = inspector.BlobStoreRootDir()
	}
	return info
}

// CheckBlobStore runs the optional blob-store health check when implemented.
func CheckBlobStore(ctx context.Context, store BlobStore) error {
	if store == nil {
		return nil
	}
	if checker, ok := store.(BlobStoreHealthChecker); ok {
		return checker.CheckBlobStore(ctx)
	}
	return nil
}
