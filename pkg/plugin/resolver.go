package plugin

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

// ResolverSetOptions describes the narrow resolver implementations bound to an
// AppContext. Nil resolver fields are replaced with no-op implementations.
type ResolverSetOptions struct {
	CI             CIResolver
	ChangeDetector ChangeDetectorResolver
	KVCache        KVCacheResolver
	BlobStore      BlobStoreResolver
}

// ResolverSet is the immutable resolver bundle bound to an AppContext.
// Consumers still read only through AppContext's narrow resolver accessors.
type ResolverSet struct {
	ci             CIResolver
	changeDetector ChangeDetectorResolver
	kvCache        KVCacheResolver
	blobStore      BlobStoreResolver
}

// NewResolverSet creates a resolver set with no-op defaults for missing
// capabilities.
func NewResolverSet(opts ResolverSetOptions) ResolverSet {
	set := ResolverSet{
		ci:             opts.CI,
		changeDetector: opts.ChangeDetector,
		kvCache:        opts.KVCache,
		blobStore:      opts.BlobStore,
	}
	if set.ci == nil {
		set.ci = noopCIResolver{}
	}
	if set.changeDetector == nil {
		set.changeDetector = noopChangeDetectorResolver{}
	}
	if set.kvCache == nil {
		set.kvCache = noopKVCacheResolver{}
	}
	if set.blobStore == nil {
		set.blobStore = noopBlobStoreResolver{}
	}
	return set
}

// NoopResolverSet returns a resolver set that rejects every capability lookup
// with ErrNoResolver.
func NoopResolverSet() ResolverSet {
	return NewResolverSet(ResolverSetOptions{})
}

// CIResolver returns the CI provider resolver. Always non-nil.
func (s ResolverSet) CIResolver() CIResolver {
	if s.ci == nil {
		return noopCIResolver{}
	}
	return s.ci
}

// ChangeDetectorResolver returns the change detector resolver. Always non-nil.
func (s ResolverSet) ChangeDetectorResolver() ChangeDetectorResolver {
	if s.changeDetector == nil {
		return noopChangeDetectorResolver{}
	}
	return s.changeDetector
}

// KVCacheResolver returns the KV cache resolver. Always non-nil.
func (s ResolverSet) KVCacheResolver() KVCacheResolver {
	if s.kvCache == nil {
		return noopKVCacheResolver{}
	}
	return s.kvCache
}

// BlobStoreResolver returns the blob store resolver. Always non-nil.
func (s ResolverSet) BlobStoreResolver() BlobStoreResolver {
	if s.blobStore == nil {
		return noopBlobStoreResolver{}
	}
	return s.blobStore
}
