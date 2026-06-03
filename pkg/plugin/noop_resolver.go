package plugin

import (
	"errors"
)

// ErrNoResolver is returned by default no-op capability lookups. Tests can
// match it via errors.Is to assert the no-resolver path.
var ErrNoResolver = errors.New("plugin resolver is not configured")

type noopCIResolver struct{}

func (noopCIResolver) ResolveCIProvider() (*ResolvedCIProvider, error) {
	return nil, ErrNoResolver
}

type noopChangeDetectorResolver struct{}

func (noopChangeDetectorResolver) ResolveChangeDetector() (ChangeDetectionProvider, error) {
	return nil, ErrNoResolver
}

type noopKVCacheResolver struct{}

func (noopKVCacheResolver) ResolveKVCacheProvider(string, ...string) (KVCacheProvider, error) {
	return nil, ErrNoResolver
}

type noopBlobStoreResolver struct{}

func (noopBlobStoreResolver) ResolveBlobStoreProvider(string, ...string) (BlobStoreProvider, error) {
	return nil, ErrNoResolver
}
