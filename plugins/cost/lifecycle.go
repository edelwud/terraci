package cost

import (
	"context"
	"time"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	costengine "github.com/edelwud/terraci/plugins/cost/internal"
)

// Initialize creates the estimator and cleans expired cache at startup.
func (p *Plugin) Initialize(_ context.Context, _ *plugin.AppContext) error {
	if p.cfg == nil || !p.cfg.Enabled {
		return nil
	}

	log.Debug("cost: initializing estimator and pricing cache")
	p.estimator = costengine.NewEstimatorFromConfig(p.cfg)
	p.estimator.CleanExpiredCache()

	entries := p.estimator.CacheEntries()
	if len(entries) == 0 {
		log.WithField("dir", p.estimator.CacheDir()).Debug("pricing cache empty")
	} else {
		for _, e := range entries {
			log.WithField("service", string(e.Service)).
				WithField("region", e.Region).
				WithField("expires_in", e.ExpiresIn.Truncate(time.Minute)).
				Debug("pricing cache")
		}
	}

	return nil
}
