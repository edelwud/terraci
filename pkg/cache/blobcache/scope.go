package blobcache

import "context"

type scopedStore interface {
	Get(ctx context.Context, key string) ([]byte, Meta, bool, error)
	Put(ctx context.Context, key string, value []byte, opts PutOptions) (Meta, error)
	Delete(ctx context.Context, key string) error
	DeleteNamespace(ctx context.Context) error
	List(ctx context.Context) ([]Object, error)
	Dir() string
}

type blobStoreScope struct {
	store     Store
	namespace string
	dir       string
}

func newBlobStoreScope(store Store, namespace string) scopedStore {
	if store == nil {
		return nil
	}
	return blobStoreScope{
		store:     store,
		namespace: namespace,
		dir:       Describe(store, "").Root,
	}
}

func (s blobStoreScope) Get(ctx context.Context, key string) (data []byte, meta Meta, ok bool, err error) {
	data, ok, meta, err = s.store.Get(ctx, s.namespace, key)
	if err != nil {
		return nil, Meta{}, false, err
	}
	return data, meta, ok, nil
}

func (s blobStoreScope) Put(ctx context.Context, key string, value []byte, opts PutOptions) (Meta, error) {
	return s.store.Put(ctx, s.namespace, key, value, PutOptions{
		ContentType: opts.ContentType,
		ExpiresAt:   cloneTimePtr(opts.ExpiresAt),
		Metadata:    cloneStringMap(opts.Metadata),
	})
}

func (s blobStoreScope) Delete(ctx context.Context, key string) error {
	return s.store.Delete(ctx, s.namespace, key)
}

func (s blobStoreScope) DeleteNamespace(ctx context.Context) error {
	return s.store.DeleteNamespace(ctx, s.namespace)
}

func (s blobStoreScope) List(ctx context.Context) ([]Object, error) {
	return s.store.List(ctx, s.namespace)
}

func (s blobStoreScope) Dir() string {
	return s.dir
}
