package policy

import (
	"context"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/policy/internal/engine"
)

// Preflight validates policy plugin prerequisites.
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	if !p.IsEnabled() {
		return nil
	}

	if err := p.Config().Validate(); err != nil {
		return err
	}

	log.WithField("opa", engine.OPAVersion()).Debug("policy: configuration validated")
	return nil
}
