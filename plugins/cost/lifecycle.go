package cost

import (
	"context"

	"github.com/edelwud/terraci/pkg/log"
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
