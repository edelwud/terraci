package update

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
	"github.com/edelwud/terraci/plugins/update/internal/registryclient"
)

type runtimeOptions struct {
	write      bool
	modulePath string
	outputFmt  string
	target     string
	bump       string
}

type updateRuntime struct {
	config   *updateengine.UpdateConfig
	registry updateengine.RegistryClient
	options  runtimeOptions
}

func newRuntime(
	ctx context.Context,
	appCtx *plugin.AppContext,
	cfg *updateengine.UpdateConfig,
	registryFactory func() updateengine.RegistryClient,
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
		runtimeConfig.Bump = opts.bump
	}
	if runtimeConfig.Target == "" {
		runtimeConfig.Target = updateengine.TargetAll
	}
	if runtimeConfig.Bump == "" {
		runtimeConfig.Bump = updateengine.BumpMinor
	}
	if err := runtimeConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	if registryFactory == nil {
		registryFactory = func() updateengine.RegistryClient {
			return registryclient.New()
		}
	}
	cacheProvider, err := registry.ResolveKVCacheProvider(runtimeConfig.CacheBackend())
	if err != nil {
		return nil, fmt.Errorf("resolve cache backend: %w", err)
	}

	cache, err := cacheProvider.NewKVCache(ctx, appCtx)
	if err != nil {
		return nil, fmt.Errorf("create cache backend %q: %w", cacheProvider.Name(), err)
	}

	cachedRegistry := updateengine.NewCachedRegistryClient(
		registryFactory(),
		cache,
		runtimeConfig.CacheNamespace(),
		runtimeConfig.CacheTTL(),
	)
	if cachedRegistry == nil {
		return nil, errors.New("failed to create cached registry client")
	}

	return &updateRuntime{
		config:   &runtimeConfig,
		registry: cachedRegistry,
		options:  opts,
	}, nil
}

func (p *Plugin) Runtime(ctx context.Context, appCtx *plugin.AppContext) (any, error) {
	return newRuntime(ctx, appCtx, p.Config(), p.registryFactory, runtimeOptions{})
}

func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext, opts runtimeOptions) (*updateRuntime, error) {
	if opts == (runtimeOptions{}) {
		return plugin.BuildRuntime[*updateRuntime](ctx, p, appCtx)
	}
	return newRuntime(ctx, appCtx, p.Config(), p.registryFactory, opts)
}
