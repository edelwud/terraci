package cost

import (
	"context"
	"time"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	costengine "github.com/edelwud/terraci/plugins/cost/internal"
)

// Initialize creates the estimator and cleans expired cache at startup.
func (p *Plugin) Initialize(_ context.Context, appCtx *plugin.AppContext) error {
	cfg := appCtx.Config()
	if cfg != nil {
		p.serviceDirRel = cfg.ServiceDir
	}

	if !p.IsEnabled() {
		return nil
	}

	// Validate config before proceeding
	if err := p.Config().Validate(); err != nil {
		log.WithError(err).Warn("cost: invalid configuration, using defaults")
	}

	log.Debug("cost: initializing estimator and pricing cache")
	estimator, err := costengine.NewEstimatorFromConfig(p.Config())
	if err != nil {
		return err
	}
	p.estimator = estimator
	p.estimator.CleanExpiredCache()

	entries := p.estimator.CacheEntries()
	if len(entries) == 0 {
		log.WithField("dir", p.estimator.CacheDir()).Debug("pricing cache empty")
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
