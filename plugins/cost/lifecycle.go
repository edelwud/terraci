package cost

import (
	"context"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
)

// Preflight validates the cost plugin configuration.
// Cache state logging happens lazily inside newRuntime when the estimator is first built.
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	if !p.IsEnabled() {
		return nil
	}

	log.Debug("cost: validating configuration")
	return validateRuntimeConfig(p.Config())
}
