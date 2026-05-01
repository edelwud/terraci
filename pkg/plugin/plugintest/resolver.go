package plugintest

import (
	"errors"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

// NoopResolver is a default-deny implementation of plugin.Resolver useful for
// tests. Tests embed it and override only the methods relevant to the case at
// hand.
type NoopResolver struct{}

// All returns no plugins.
func (NoopResolver) All() []plugin.Plugin { return nil }

// GetPlugin returns nothing.
func (NoopResolver) GetPlugin(string) (plugin.Plugin, bool) { return nil, false }

// ResolveCIProvider rejects with a generic unsupported error.
func (NoopResolver) ResolveCIProvider() (*plugin.ResolvedCIProvider, error) {
	return nil, errors.New("plugintest: CI provider resolution not configured")
}

// ResolveChangeDetector rejects with a generic unsupported error.
func (NoopResolver) ResolveChangeDetector() (plugin.ChangeDetectionProvider, error) {
	return nil, errors.New("plugintest: change detector resolution not configured")
}

// ResolveKVCacheProvider rejects with a generic unsupported error.
func (NoopResolver) ResolveKVCacheProvider(string) (plugin.KVCacheProvider, error) {
	return nil, errors.New("plugintest: kv cache provider resolution not configured")
}

// ResolveBlobStoreProvider rejects with a generic unsupported error.
func (NoopResolver) ResolveBlobStoreProvider(string) (plugin.BlobStoreProvider, error) {
	return nil, errors.New("plugintest: blob store provider resolution not configured")
}

// CollectContributions returns no contributions.
func (NoopResolver) CollectContributions(*plugin.AppContext) []*pipeline.Contribution {
	return nil
}

// PreflightsForStartup returns no preflightables.
func (NoopResolver) PreflightsForStartup() []plugin.Preflightable { return nil }
