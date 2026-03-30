package gitlab

import (
	"context"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	gitlabci "github.com/edelwud/terraci/plugins/gitlab/internal"
)

// Preflight detects MR context when running inside GitLab CI.
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	if !p.DetectEnv() {
		return nil
	}

	mrCtx := gitlabci.DetectMRContext()
	if mrCtx.InMR {
		log.WithField("mr", mrCtx.MRIID).Debug("gitlab: MR context detected")
	} else {
		log.Debug("gitlab: CI detected but not in MR pipeline")
	}

	return nil
}

func (p *Plugin) Initialize(ctx context.Context, appCtx *plugin.AppContext) error {
	return p.Preflight(ctx, appCtx)
}
