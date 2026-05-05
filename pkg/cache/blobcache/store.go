package blobcache

import (
	"context"
	"io"
	"time"
)

// Store stores opaque binary objects addressed by namespace + key.
//
// Consumers own key layout, serialization, TTL policy, and stale fallback
// semantics. Backends only persist bytes and metadata.
//
// Thread-safety: implementations MUST be safe for concurrent use across
// goroutines, including Put + Get + Delete on the same (namespace, key)
// from multiple goroutines simultaneously. The contract test suite at
// pkg/cache/blobcache/contracttest exercises this guarantee — register a
// new backend against it before merging.
//
// Cross-process safety is best-effort and backend-specific. The bundled
// diskblob backend serializes per-key writes via an in-process keyed
// mutex; multi-process protection requires either separate root_dir
// values or external coordination (advisory file locks, etc.).
type Store interface {
	Get(ctx context.Context, namespace, key string) ([]byte, bool, Meta, error)
	Put(ctx context.Context, namespace, key string, value []byte, opts PutOptions) (Meta, error)
	Open(ctx context.Context, namespace, key string) (io.ReadCloser, bool, Meta, error)
	PutStream(ctx context.Context, namespace, key string, r io.Reader, opts PutOptions) (Meta, error)
	Delete(ctx context.Context, namespace, key string) error
	DeleteNamespace(ctx context.Context, namespace string) error
	List(ctx context.Context, namespace string) ([]Object, error)
}

// PutOptions controls how an entry is persisted by a Store.
type PutOptions struct {
	ContentType string
	ExpiresAt   *time.Time
	Metadata    map[string]string
}

// Meta describes a stored blob object.
type Meta struct {
	Size        int64
	UpdatedAt   time.Time
	ExpiresAt   *time.Time
	ETag        string
	ContentType string
	Metadata    map[string]string
}

// Object describes a listed blob object.
type Object struct {
	Key  string
	Meta Meta
}

// Info describes optional backend diagnostics exposed by a blob store.
type Info struct {
	Backend                 string
	Root                    string
	SupportsList            bool
	SupportsStream          bool
	SupportsDeleteNamespace bool
}

// Inspector exposes optional store details for diagnostics.
type Inspector interface {
	BlobStoreRootDir() string
}

// Describer exposes optional diagnostics for a blob store backend.
type Describer interface {
	DescribeBlobStore() Info
}

// HealthChecker exposes an optional health check for a blob store backend.
type HealthChecker interface {
	CheckBlobStore(ctx context.Context) error
}

// Describe returns the optional backend diagnostics exposed by a blob store,
// applying a fallback backend name when the store omits one.
func Describe(store Store, fallbackBackend string) Info {
	if store == nil {
		return Info{Backend: fallbackBackend}
	}

	if describer, ok := store.(Describer); ok {
		info := describer.DescribeBlobStore()
		if info.Backend == "" {
			info.Backend = fallbackBackend
		}
		if info.Root != "" || info.Backend != "" || info.SupportsList || info.SupportsStream || info.SupportsDeleteNamespace {
			return info
		}
	}

	info := Info{Backend: fallbackBackend}
	if inspector, ok := store.(Inspector); ok {
		info.Root = inspector.BlobStoreRootDir()
	}
	return info
}

// Check runs the optional blob store health check when implemented.
func Check(ctx context.Context, store Store) error {
	if store == nil {
		return nil
	}
	if checker, ok := store.(HealthChecker); ok {
		return checker.CheckBlobStore(ctx)
	}
	return nil
}
