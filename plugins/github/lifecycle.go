package github

import (
	"context"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	githubci "github.com/edelwud/terraci/plugins/github/internal"
)

// Initialize detects PR context at startup.
func (p *Plugin) Initialize(_ context.Context, _ *plugin.AppContext) error {
	p.inCI = p.DetectEnv()
	if !p.inCI {
		return nil
	}

	p.prCtx = githubci.DetectPRContext()
	if p.prCtx.InPR {
		log.WithField("pr", p.prCtx.PRNumber).Debug("github: PR context detected")
	} else {
		log.Debug("github: Actions detected but not in PR workflow")
	}

	return nil
}
