package github

import (
	"context"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
	prpkg "github.com/edelwud/terraci/plugins/github/internal/pr"
)

// Preflight validates the loaded plugin config and detects PR context when
// running inside GitHub Actions. Validation runs before the env-detection
// branch so misshapen configs fail fast even on local generate runs.
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	if cfg := p.Config(); cfg != nil {
		if err := cfg.Validate(); err != nil {
			return err
		}
	}

	if !p.DetectEnv() {
		return nil
	}

	prCtx := prpkg.DetectContext()
	if prCtx.InPR {
		log.WithField("pr", prCtx.PRNumber).Debug("github: PR context detected")
	} else {
		log.Debug("github: Actions detected but not in PR workflow")
	}

	return nil
}
