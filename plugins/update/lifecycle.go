package update

import (
	"context"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
)

// Preflight validates update plugin configuration.
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
