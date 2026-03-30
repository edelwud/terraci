package github

import (
	"context"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	githubci "github.com/edelwud/terraci/plugins/github/internal"
)

// Preflight detects PR context when running inside GitHub Actions.
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	if !p.DetectEnv() {
		return nil
	}

	prCtx := githubci.DetectPRContext()
	if prCtx.InPR {
		log.WithField("pr", prCtx.PRNumber).Debug("github: PR context detected")
	} else {
		log.Debug("github: Actions detected but not in PR workflow")
	}

	return nil
}
