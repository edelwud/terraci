package git

import (
	"context"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
	gitclient "github.com/edelwud/terraci/plugins/git/internal"
)

// Preflight verifies whether the workdir is a git repository.
func (p *Plugin) Preflight(_ context.Context, appCtx *plugin.AppContext) error {
	client := gitclient.NewClient(appCtx.WorkDir())
	if !client.IsGitRepo() {
		log.Debug("git: not a git repository, change detection disabled")
		return nil
	}

	log.WithField("branch", client.GetDefaultBranch()).Debug("git: repository detected")

	return nil
}
func (p *Plugin) resolveRef(baseRef string, client *gitclient.Client) string {
	if baseRef != "" {
		return baseRef
	}
	if defaultRef := client.GetDefaultBranch(); defaultRef != "" {
		return defaultRef
	}
	return "main"
}
