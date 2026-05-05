package gitlab

import (
	"context"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	mrpkg "github.com/edelwud/terraci/plugins/gitlab/internal/mr"
)

// Preflight validates the loaded plugin config and detects MR context when
// running inside GitLab CI. Validation runs before the env-detection branch
// so misshapen configs fail fast even on local generate runs.
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	if cfg := p.Config(); cfg != nil {
		if err := cfg.Validate(); err != nil {
			return err
		}
	}

	if !p.DetectEnv() {
		return nil
	}

	mrCtx := mrpkg.DetectContext()
	if mrCtx.InMR {
		log.WithField("mr", mrCtx.MRIID).Debug("gitlab: MR context detected")
	} else {
		log.Debug("gitlab: CI detected but not in MR pipeline")
	}

	return nil
}
