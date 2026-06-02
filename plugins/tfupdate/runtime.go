package tfupdate

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/lockfile"
	tfregistry "github.com/edelwud/terraci/plugins/tfupdate/internal/registry"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registryclient"
)

type updateRuntime struct {
	config     *tfupdateengine.UpdateConfig
	registry   tfregistry.Client
	downloader lockfile.Downloader
}

func newRuntime(
	ctx context.Context,
	appCtx *plugin.AppContext,
	cfg *tfupdateengine.UpdateConfig,
	registryFactory func() tfregistry.Client,
) (*updateRuntime, error) {
	if cfg == nil {
		return nil, errors.New("update configuration is not set")
	}

	runtimeConfig := *cfg
	if runtimeConfig.Target == "" {
		runtimeConfig.Target = tfupdateengine.TargetAll
	}
	if err := runtimeConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	if registryFactory == nil {
		registryFactory = func() tfregistry.Client {
			return registryclient.New()
		}
	}
	cacheProvider, err := appCtx.KVCacheResolver().ResolveKVCacheProvider(
		runtimeConfig.MetadataCacheBackend(),
		"set extensions.tfupdate.cache.metadata.backend explicitly",
	)
	if err != nil {
		return nil, fmt.Errorf("resolve cache backend: %w", err)
	}

	cache, err := cacheProvider.NewKVCache(ctx, appCtx)
	if err != nil {
		return nil, fmt.Errorf("create cache backend %q: %w", cacheProvider.Name(), err)
	}

	cachedRegistry := tfregistry.NewCachedClient(
		registryFactory(),
		cache,
		runtimeConfig.MetadataCacheNamespace(),
		runtimeConfig.MetadataCacheTTL(),
	)
	if cachedRegistry == nil {
		return nil, errors.New("failed to create cached registry client")
	}

	blobProvider, err := appCtx.BlobStoreResolver().ResolveBlobStoreProvider(
		runtimeConfig.ArtifactCacheBackend(),
		"set extensions.tfupdate.cache.artifacts.backend explicitly",
	)
	if err != nil {
		return nil, fmt.Errorf("resolve artifact cache backend: %w", err)
	}

	blobStore, err := blobProvider.NewBlobStore(ctx, appCtx, plugin.BlobStoreOptions{})
	if err != nil {
		return nil, fmt.Errorf("create artifact cache backend %q: %w", blobProvider.Name(), err)
	}

	downloader := lockfile.NewBlobCachingDownloader(
		lockfile.NewHTTPDownloader(),
		blobStore,
		runtimeConfig.ArtifactCacheNamespace(),
	)

	return &updateRuntime{
		config:     &runtimeConfig,
		registry:   cachedRegistry,
		downloader: downloader,
	}, nil
}

// runtime returns the typed plugin runtime used by tfupdate use-cases.
func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext) (*updateRuntime, error) {
	return newRuntime(ctx, appCtx, p.Config(), p.registryFactory)
}
