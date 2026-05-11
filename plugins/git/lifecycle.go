package git

import (
	"context"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/git/internal/gitclient"
)

// Preflight verifies whether the workdir is a git repository.
func (p *Plugin) Preflight(_ context.Context, appCtx *plugin.AppContext) error {
	client := gitclient.NewClient(appCtx.WorkDir())
	if !client.IsGitRepo() {
		log.Debug("git: not a git repository, change detection disabled")
		return nil
	}

	log.WithField("base_ref", client.ResolveBaseRef("")).Debug("git: repository detected")

	if shallow, err := client.IsShallow(); err == nil && shallow {
		log.Warn("git: shallow clone detected; --changed-only requires full history. Configure the CI checkout with full fetch depth or fetch the base branch/history before invoking change detection.")
	}

	return nil
}
