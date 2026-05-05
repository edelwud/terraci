package plugin

// CommandLookup is the framework-side lookup used only to bind a cobra
// command callback back to its command-scoped plugin instance.
type CommandLookup interface {
	// GetPlugin returns a plugin by name from the current command-scoped set.
	GetPlugin(name string) (Plugin, bool)
}

// Resolver is the plugin-visible capability resolver. It deliberately does not
// expose raw plugin enumeration or name lookup; plugins communicate through
// typed capabilities instead of concrete plugin names.
type Resolver interface {
	// ResolveCIProvider returns the active CI provider in the current set.
	ResolveCIProvider() (*ResolvedCIProvider, error)

	// ResolveChangeDetector returns the active change-detection provider.
	ResolveChangeDetector() (ChangeDetectionProvider, error)

	// ResolveKVCacheProvider returns the named KV cache backend provider.
	// Pass an optional configPathHint to make ambiguous-backend errors point at
	// the calling feature's exact config field.
	ResolveKVCacheProvider(name string, configPathHint ...string) (KVCacheProvider, error)

	// ResolveBlobStoreProvider returns the named blob store backend provider.
	// Pass an optional configPathHint to make ambiguous-backend errors point at
	// the calling feature's exact config field.
	ResolveBlobStoreProvider(name string, configPathHint ...string) (BlobStoreProvider, error)
}
