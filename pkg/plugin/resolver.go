package plugin

// CommandLookup is the framework-side lookup used only to bind a cobra
// command callback back to its command-scoped plugin instance.
type CommandLookup interface {
	// GetPlugin returns a plugin by name from the current command-scoped set.
	GetPlugin(name string) (Plugin, bool)
}

// CIResolver resolves the active CI provider in the current plugin set.
type CIResolver interface {
	ResolveCIProvider() (*ResolvedCIProvider, error)
}

// ChangeDetectorResolver resolves the active change-detection provider.
type ChangeDetectorResolver interface {
	ResolveChangeDetector() (ChangeDetectionProvider, error)
}

// KVCacheResolver resolves named KV cache backend providers. Pass an optional
// configPathHint to make ambiguous-backend errors point at the calling
// feature's exact config field.
type KVCacheResolver interface {
	ResolveKVCacheProvider(name string, configPathHint ...string) (KVCacheProvider, error)
}

// BlobStoreResolver resolves named blob store backend providers. Pass an
// optional configPathHint to make ambiguous-backend errors point at the calling
// feature's exact config field.
type BlobStoreResolver interface {
	ResolveBlobStoreProvider(name string, configPathHint ...string) (BlobStoreProvider, error)
}

// Resolver is the framework resolver implementation contract. Plugin code
// should prefer AppContext's narrow resolver accessors instead of depending on
// this aggregate interface directly.
type Resolver interface {
	CIResolver
	ChangeDetectorResolver
	KVCacheResolver
	BlobStoreResolver
}
