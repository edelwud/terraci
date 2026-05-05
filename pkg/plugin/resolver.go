package plugin

import "github.com/edelwud/terraci/pkg/pipeline"

// Lookup is the read-side of the command-scoped plugin set: enumeration
// and name lookup. External callers (e.g. tests for non-capability code) only
// need this slice of Resolver and can mock it cheaply.
type Lookup interface {
	// All returns every plugin in the current command-scoped set.
	All() []Plugin

	// GetPlugin returns a plugin by name from the current set.
	GetPlugin(name string) (Plugin, bool)
}

// CapabilityResolver resolves the canonical capability backings — CI provider,
// change detector, named cache backends. Each method returns a typed handle
// or a sentinel error explaining why no plugin satisfies the capability.
type CapabilityResolver interface {
	// ResolveCIProvider returns the active CI provider in the current set.
	ResolveCIProvider() (*ResolvedCIProvider, error)

	// ResolveChangeDetector returns the active change-detection provider.
	ResolveChangeDetector() (ChangeDetectionProvider, error)

	// ResolveKVCacheProvider returns the named KV cache backend provider.
	ResolveKVCacheProvider(name string) (KVCacheProvider, error)

	// ResolveBlobStoreProvider returns the named blob store backend provider.
	ResolveBlobStoreProvider(name string) (BlobStoreProvider, error)
}

// LifecycleSource exposes plugin-collection hooks the framework uses to drive
// pipeline construction and startup preflight.
type LifecycleSource interface {
	// CollectContributions gathers pipeline contributions from enabled
	// PipelineContributor plugins.
	CollectContributions(ctx *AppContext) []*pipeline.Contribution

	// PreflightsForStartup returns enabled plugins that participate in
	// framework startup preflight for the current configuration state.
	PreflightsForStartup() []Preflightable
}

// Resolver is the full command-scoped plugin surface exposed to plugins
// through AppContext. It composes Lookup, CapabilityResolver, and
// LifecycleSource — split apart so test mocks only need to implement the
// sub-interface their callee depends on.
type Resolver interface {
	Lookup
	CapabilityResolver
	LifecycleSource
}
