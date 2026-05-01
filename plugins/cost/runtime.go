package cost

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
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

	cache, err := resolveBlobCache(ctx, appCtx, cfg)
	if err != nil {
		return nil, err
	}

	estimator, err := engine.NewEstimatorFromConfig(cfg, cache)
	if err != nil {
		return nil, fmt.Errorf("create cost estimator: %w", err)
	}

	logCacheState(ctx, estimator)
	estimator.Cache().CleanExpired(ctx)

	return &costRuntime{estimator: estimator}, nil
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

// resolveBlobCache resolves the underlying blob store and wraps it in a blobcache.Cache
// configured with the plugin's namespace and TTL settings.
func resolveBlobCache(ctx context.Context, appCtx *plugin.AppContext, cfg *model.CostConfig) (*blobcache.Cache, error) {
	blobProvider, err := appCtx.Resolver().ResolveBlobStoreProvider(cfg.BlobCacheBackend())
	if err != nil {
		return nil, fmt.Errorf("resolve blob backend: %w", err)
	}

	blobStore, err := blobProvider.NewBlobStore(ctx, appCtx, plugin.BlobStoreOptions{})
	if err != nil {
		return nil, fmt.Errorf("create blob backend %q: %w", blobProvider.Name(), err)
	}
	if err := blobcache.Check(ctx, blobStore); err != nil {
		return nil, fmt.Errorf("check blob backend %q: %w", blobProvider.Name(), err)
	}

	info := blobcache.Describe(blobStore, blobProvider.Name())
	log.WithField("backend", info.Backend).
		WithField("root", info.Root).
		Debug("cost: resolved blob backend")

	return blobcache.New(blobStore, cfg.BlobCacheNamespace(), cfg.CacheTTLDuration()), nil
}

// logCacheState logs the current pricing cache entries for diagnostic visibility.
// Called once after the estimator is built.
func logCacheState(ctx context.Context, e *engine.Estimator) {
	cache := e.Cache()
	dir := cache.Dir()
	if dir == "" {
		return
	}

	entries := cache.Entries(ctx)

	log.WithField("dir", dir).
		WithField("ttl", cache.TTL().String()).
		WithField("entries", len(entries)).
		Debug("cost: pricing cache state")

	for _, entry := range entries {
		l := log.WithField("service", entry.Service.Name).
			WithField("region", entry.Region).
			WithField("age", entry.Age.Round(time.Second).String())

		if entry.ExpiresIn < 0 {
			l.WithField("expired_by", (-entry.ExpiresIn).Round(time.Second).String()).
				Debug("cost: cache entry (expired)")
		} else {
			l.WithField("expires_in", entry.ExpiresIn.Round(time.Second).String()).
				Debug("cost: cache entry")
		}
	}
}
