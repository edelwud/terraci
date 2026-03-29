package policy

import (
	"context"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

// Initialize validates OPA availability at startup.
func (p *Plugin) Initialize(_ context.Context, appCtx *plugin.AppContext) error {
	p.serviceDirRel = appCtx.Config.ServiceDir

	if !p.IsConfigured() {
		return nil
	}

	log.WithField("opa", policyengine.OPAVersion()).Debug("policy: OPA engine available")

	// Validate policy sources are configured
	if len(p.Config().Sources) == 0 {
		log.Warn("policy: enabled but no sources configured")
	}

	return nil
}
