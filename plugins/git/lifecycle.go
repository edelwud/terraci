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

func (p *Plugin) getClient() *gitclient.Client {
	if !p.isRepo {
		return nil
	}
	return p.client
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
