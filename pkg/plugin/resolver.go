package plugin

import "github.com/edelwud/terraci/pkg/pipeline"

// Resolver is the command-scoped plugin lookup and capability resolution surface
// exposed to plugins through AppContext. Plugins should never construct one;
// the framework binds a Resolver to the AppContext for each command run.
type Resolver interface {
	// All returns every plugin in the current command-scoped set.
	All() []Plugin

	// GetPlugin returns a plugin by name from the current set.
	GetPlugin(name string) (Plugin, bool)

	// ResolveCIProvider returns the active CI provider in the current set.
	ResolveCIProvider() (*ResolvedCIProvider, error)

	// ResolveChangeDetector returns the active change-detection provider.
	ResolveChangeDetector() (ChangeDetectionProvider, error)

	// ResolveKVCacheProvider returns the named KV cache backend provider.
	ResolveKVCacheProvider(name string) (KVCacheProvider, error)

	// ResolveBlobStoreProvider returns the named blob store backend provider.
	ResolveBlobStoreProvider(name string) (BlobStoreProvider, error)

	// CollectContributions gathers pipeline contributions from enabled
	// PipelineContributor plugins.
	CollectContributions(ctx *AppContext) []*pipeline.Contribution

	// PreflightsForStartup returns enabled plugins that participate in
	// framework startup preflight for the current configuration state.
	PreflightsForStartup() []Preflightable
}
