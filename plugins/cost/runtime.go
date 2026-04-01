package cost

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/plugins/cost/internal/engine"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

type costRuntime struct {
	estimator *engine.Estimator
}

func newRuntime(ctx context.Context, appCtx *plugin.AppContext, cfg *model.CostConfig) (*costRuntime, error) {
	if err := validateRuntimeConfig(cfg); err != nil {
		return nil, err
	}

	cache, _, err := resolveBlobCache(ctx, appCtx, cfg)
	if err != nil {
		return nil, err
	}

	estimator, err := engine.NewEstimatorFromConfigWithBlobCache(cfg, cache)
	if err != nil {
		return nil, fmt.Errorf("create cost estimator: %w", err)
	}

	return &costRuntime{estimator: estimator}, nil
}

func newRuntimeWithEstimator(estimator *engine.Estimator) *costRuntime {
	return &costRuntime{estimator: estimator}
}

// Runtime implements plugin.RuntimeProvider and serves as the reference lazy
// runtime pattern for runtime-heavy plugins in TerraCi.
func (p *Plugin) Runtime(ctx context.Context, appCtx *plugin.AppContext) (any, error) {
	return newRuntime(ctx, appCtx, p.Config())
}

func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext) (*costRuntime, error) {
	return plugin.BuildRuntime[*costRuntime](ctx, p, appCtx)
}

func validateRuntimeConfig(cfg *model.CostConfig) error {
	if cfg == nil {
		return errors.New("cost estimation is not configured")
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid cost configuration: %w", err)
	}

	return nil
}

func resolveBlobStore(ctx context.Context, appCtx *plugin.AppContext, cfg *model.CostConfig) (plugin.BlobStore, plugin.BlobStoreInfo, error) {
	blobProvider, err := registry.ResolveBlobStoreProvider(cfg.BlobCacheBackend())
	if err != nil {
		return nil, plugin.BlobStoreInfo{}, fmt.Errorf("resolve blob backend: %w", err)
	}

	blobStore, err := blobProvider.NewBlobStore(ctx, appCtx)
	if err != nil {
		return nil, plugin.BlobStoreInfo{}, fmt.Errorf("create blob backend %q: %w", blobProvider.Name(), err)
	}
	if err := plugin.CheckBlobStore(ctx, blobStore); err != nil {
		return nil, plugin.BlobStoreInfo{}, fmt.Errorf("check blob backend %q: %w", blobProvider.Name(), err)
	}

	info := plugin.DescribeBlobStore(blobStore, blobProvider.Name())
	log.WithField("backend", info.Backend).
		WithField("root", info.Root).
		Debug("cost: resolved blob backend")

	return blobStore, info, nil
}

func resolveBlobCache(ctx context.Context, appCtx *plugin.AppContext, cfg *model.CostConfig) (*blobcache.Cache, plugin.BlobStoreInfo, error) {
	blobStore, info, err := resolveBlobStore(ctx, appCtx, cfg)
	if err != nil {
		return nil, plugin.BlobStoreInfo{}, err
	}

	return blobcache.New(blobStore, cfg.BlobCacheNamespace(), engine.CacheTTLFromConfig(cfg)), info, nil
}
