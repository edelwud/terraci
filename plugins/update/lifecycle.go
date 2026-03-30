package update

import (
	"context"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

// Initialize sets up the registry client.
func (p *Plugin) Initialize(_ context.Context, _ *plugin.AppContext) error {
	if !p.IsEnabled() {
		return nil
	}

	if err := p.Config().Validate(); err != nil {
		log.WithError(err).Warn("update: invalid configuration, using defaults")
	}

	log.Debug("update: initializing registry client")
	p.registry = updateengine.NewRegistryClient()

	return nil
}
