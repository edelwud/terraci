package git

import (
	"context"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	gitclient "github.com/edelwud/terraci/plugins/git/internal"
)

// Initialize verifies the git repository and caches the client at startup.
func (p *Plugin) Initialize(_ context.Context, appCtx *plugin.AppContext) error {
	p.client = gitclient.NewClient(appCtx.WorkDir)
	p.isRepo = p.client.IsGitRepo()

	if !p.isRepo {
		log.Debug("git: not a git repository, change detection disabled")
		return nil
	}

	p.defaultRef = p.client.GetDefaultBranch()
	log.WithField("branch", p.defaultRef).Debug("git: repository detected")

	return nil
}

func (p *Plugin) getClient(appCtx *plugin.AppContext) *gitclient.Client {
	if p.client != nil && p.isRepo {
		return p.client
	}
	// Fallback if Initialize was not called
	c := gitclient.NewClient(appCtx.WorkDir)
	if !c.IsGitRepo() {
		return nil
	}
	return c
}

func (p *Plugin) resolveRef(baseRef string) string {
	if baseRef != "" {
		return baseRef
	}
	if p.defaultRef != "" {
		return p.defaultRef
	}
	return "main"
}
