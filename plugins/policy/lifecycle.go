package policy

import (
	"context"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

// Preflight validates policy plugin prerequisites.
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	if !p.IsEnabled() {
		return nil
	}

	log.WithField("opa", policyengine.OPAVersion()).Debug("policy: OPA engine available")

	// Validate policy sources are configured
	if len(p.Config().Sources) == 0 {
		log.Warn("policy: enabled but no sources configured")
	}

	return nil
}
