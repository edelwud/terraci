package gitlab

import (
	"context"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	gitlabci "github.com/edelwud/terraci/plugins/gitlab/internal"
)

// Initialize detects MR context at startup.
func (p *Plugin) Initialize(_ context.Context, _ *plugin.AppContext) error {
	p.inCI = p.DetectEnv()
	if !p.inCI {
		return nil
	}

	p.mrCtx = gitlabci.DetectMRContext()
	if p.mrCtx.InMR {
		log.WithField("mr", p.mrCtx.MRIID).Debug("gitlab: MR context detected")
	} else {
		log.Debug("gitlab: CI detected but not in MR pipeline")
	}

	return nil
}
