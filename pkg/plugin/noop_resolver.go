package plugin

import (
	"errors"
)

// NoopResolver is the default-deny Resolver. It is bound to AppContext when
// no real resolver is supplied (so plugins may always call ctx.Resolver()
// without nil-checks) and is also intended for tests: embed it and override
// only the methods relevant to the case at hand.
type NoopResolver struct{}

// ErrNoResolver is returned by NoopResolver capability lookups. Tests can
// match it via errors.Is to assert the no-resolver path.
var ErrNoResolver = errors.New("plugin resolver is not configured")

// ResolveCIProvider rejects with ErrNoResolver.
func (NoopResolver) ResolveCIProvider() (*ResolvedCIProvider, error) {
	return nil, ErrNoResolver
}

// ResolveChangeDetector rejects with ErrNoResolver.
func (NoopResolver) ResolveChangeDetector() (ChangeDetectionProvider, error) {
	return nil, ErrNoResolver
}

// ResolveKVCacheProvider rejects with ErrNoResolver.
func (NoopResolver) ResolveKVCacheProvider(string, ...string) (KVCacheProvider, error) {
	return nil, ErrNoResolver
}

// ResolveBlobStoreProvider rejects with ErrNoResolver.
func (NoopResolver) ResolveBlobStoreProvider(string, ...string) (BlobStoreProvider, error) {
	return nil, ErrNoResolver
}
