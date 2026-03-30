package cost

import (
	"context"
	"time"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
)

// Initialize creates the estimator and cleans expired cache at startup.
func (p *Plugin) Initialize(_ context.Context, _ *plugin.AppContext) error {
	if !p.IsEnabled() {
		return nil
	}

	log.Debug("cost: validating runtime and pricing cache")
	runtime, err := newRuntime(p.Config())
	if err != nil {
		return err
	}
	runtime.estimator.CleanExpiredCache()

	entries := runtime.estimator.CacheEntries()
	if len(entries) == 0 {
		log.WithField("dir", runtime.estimator.CacheDir()).Debug("pricing cache empty")
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
