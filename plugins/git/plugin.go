// Package git provides the Git change detection plugin for TerraCi.
package git

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	gitclient "github.com/edelwud/terraci/plugins/git/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

// Plugin is the Git change detection plugin.
type Plugin struct {
	client     *gitclient.Client
	defaultRef string
	isRepo     bool
}

func (p *Plugin) Name() string        { return "git" }
func (p *Plugin) Description() string { return "Git change detection for incremental pipelines" }

// Initializable — verify git repo and cache client at startup

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

// ChangeDetectionProvider

func (p *Plugin) DetectChangedModules(_ context.Context, appCtx *plugin.AppContext, baseRef string, moduleIndex *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	client := p.getClient(appCtx)
	if client == nil {
		return nil, nil, fmt.Errorf("not a git repository: %s", appCtx.WorkDir)
	}

	ref := p.resolveRef(baseRef)
	detector := gitclient.NewChangedModulesDetector(client, moduleIndex, appCtx.WorkDir)
	return detector.DetectChangedModulesVerbose(ref)
}

func (p *Plugin) DetectChangedLibraries(_ context.Context, appCtx *plugin.AppContext, baseRef string, libraryPaths []string) ([]string, error) {
	client := p.getClient(appCtx)
	if client == nil {
		return nil, nil
	}

	ref := p.resolveRef(baseRef)
	detector := gitclient.NewChangedModulesDetector(client, discovery.NewModuleIndex(nil), appCtx.WorkDir)
	return detector.DetectChangedLibraryModules(ref, libraryPaths)
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
