package tfupdate

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
	pluginregistry "github.com/edelwud/terraci/pkg/plugin/registry"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/lockfile"
	tfregistry "github.com/edelwud/terraci/plugins/tfupdate/internal/registry"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registryclient"
)

type runtimeOptions struct {
	write         bool
	modulePath    string
	outputFmt     string
	target        string
	bump          string
	pin           bool
	timeout       string
	lockPlatforms []string
}

type updateRuntime struct {
	config     *tfupdateengine.UpdateConfig
	registry   tfregistry.Client
	downloader lockfile.Downloader
	options    runtimeOptions
}

func newRuntime(
	ctx context.Context,
	appCtx *plugin.AppContext,
	cfg *tfupdateengine.UpdateConfig,
	registryFactory func() tfregistry.Client,
	opts runtimeOptions,
) (*updateRuntime, error) {
	if cfg == nil {
		return nil, errors.New("update configuration is not set")
	}

	runtimeConfig := *cfg
	if opts.target != "" {
		runtimeConfig.Target = opts.target
	}
	if opts.bump != "" {
		runtimeConfig.Policy.Bump = opts.bump
	}
	if opts.pin {
		runtimeConfig.Policy.Pin = true
	}
	if opts.timeout != "" {
		runtimeConfig.Timeout = opts.timeout
	}
	if len(opts.lockPlatforms) > 0 {
		runtimeConfig.Lock.Platforms = opts.lockPlatforms
	}
	if runtimeConfig.Target == "" {
		runtimeConfig.Target = tfupdateengine.TargetAll
	}
	if err := runtimeConfig.ValidateRuntime(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	if registryFactory == nil {
		registryFactory = func() tfregistry.Client {
			return registryclient.New()
		}
	}
	cacheProvider, err := pluginregistry.ResolveKVCacheProvider(runtimeConfig.MetadataCacheBackend())
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

	blobProvider, err := pluginregistry.ResolveBlobStoreProvider(runtimeConfig.ArtifactCacheBackend())
	if err != nil {
		return nil, fmt.Errorf("resolve artifact cache backend: %w", err)
	}

	blobStore, err := blobProvider.NewBlobStore(ctx, appCtx)
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
		options:    opts,
	}, nil
}

func (o runtimeOptions) isZero() bool {
	return !o.write && o.modulePath == "" && o.outputFmt == "" &&
		o.target == "" && o.bump == "" && !o.pin &&
		o.timeout == "" && len(o.lockPlatforms) == 0
}

func (p *Plugin) Runtime(ctx context.Context, appCtx *plugin.AppContext) (any, error) {
	return newRuntime(ctx, appCtx, p.Config(), p.registryFactory, runtimeOptions{})
}

func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext, opts runtimeOptions) (*updateRuntime, error) {
	if opts.isZero() {
		return plugin.BuildRuntime[*updateRuntime](ctx, p, appCtx)
	}
	return newRuntime(ctx, appCtx, p.Config(), p.registryFactory, opts)
}
