package cost

import (
	"context"
	"time"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// Preflight validates runtime configuration and logs pricing cache state.
func (p *Plugin) Preflight(ctx context.Context, appCtx *plugin.AppContext) error {
	if !p.IsEnabled() {
		return nil
	}

	log.Debug("cost: validating runtime and pricing cache")
	cfg := p.Config()
	if err := validateRuntimeConfig(cfg); err != nil {
		return err
	}

	cache, info, err := resolveBlobCache(ctx, appCtx, cfg)
	if err != nil {
		return err
	}
	inspector := pricing.NewCacheInspector(cache)
	if err := cache.CleanExpired(ctx); err != nil {
		log.WithError(err).Debug("pricing cache cleanup failed")
	}

	entries := inspector.Entries(ctx)
	if len(entries) == 0 {
		log.WithField("dir", inspector.Dir()).
			WithField("root", info.Root).
			Debug("pricing cache empty")
	} else {
		for _, e := range entries {
			log.WithField("service", e.Service.String()).
				WithField("region", e.Region).
				WithField("expires_in", e.ExpiresIn.Truncate(time.Minute)).
				Debug("pricing cache")
		}
	}

	return nil
}
