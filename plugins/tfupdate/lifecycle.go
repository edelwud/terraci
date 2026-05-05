package tfupdate

import (
	"context"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
)

// Preflight validates tfupdate plugin configuration.
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	if !p.IsEnabled() {
		return nil
	}

	if err := p.Config().Validate(); err != nil {
		return err
	}

	log.Debug("update: configuration validated")

	return nil
}
