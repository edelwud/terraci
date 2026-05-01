package plugin

import (
	"errors"

	"github.com/edelwud/terraci/pkg/pipeline"
)

// noopResolver is the default Resolver bound to AppContext when none is
// supplied. Lookups return nothing and resolution methods return a sentinel
// error so plugins can always call ctx.Resolver() without nil-checks.
type noopResolver struct{}

var errNoResolver = errors.New("plugin resolver is not configured")

func (noopResolver) All() []Plugin                   { return nil }
func (noopResolver) GetPlugin(string) (Plugin, bool) { return nil, false }
func (noopResolver) ResolveCIProvider() (*ResolvedCIProvider, error) {
	return nil, errNoResolver
}
func (noopResolver) ResolveChangeDetector() (ChangeDetectionProvider, error) {
	return nil, errNoResolver
}
func (noopResolver) ResolveKVCacheProvider(string) (KVCacheProvider, error) {
	return nil, errNoResolver
}
func (noopResolver) ResolveBlobStoreProvider(string) (BlobStoreProvider, error) {
	return nil, errNoResolver
}
func (noopResolver) CollectContributions(*AppContext) []*pipeline.Contribution { return nil }
func (noopResolver) PreflightsForStartup() []Preflightable                     { return nil }
