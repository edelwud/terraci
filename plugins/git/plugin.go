// Package git provides the Git change detection plugin for TerraCi.
package git

import (
	"context"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/git/gitops"
)

func init() {
	plugin.Register(&Plugin{})
}

// Plugin is the Git change detection plugin.
type Plugin struct{}

func (p *Plugin) Name() string        { return "git" }
func (p *Plugin) Description() string { return "Git change detection for incremental pipelines" }

// ChangeDetectionProvider

func (p *Plugin) DetectChangedModules(_ context.Context, appCtx *plugin.AppContext, baseRef string, moduleIndex *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	client := gitops.NewClient(appCtx.WorkDir)
	if !client.IsGitRepo() {
		return nil, nil, nil
	}

	ref := baseRef
	if ref == "" {
		ref = client.GetDefaultBranch()
	}

	detector := gitops.NewChangedModulesDetector(client, moduleIndex, appCtx.WorkDir)
	return detector.DetectChangedModulesVerbose(ref)
}

func (p *Plugin) DetectChangedLibraries(_ context.Context, appCtx *plugin.AppContext, baseRef string, libraryPaths []string) ([]string, error) {
	client := gitops.NewClient(appCtx.WorkDir)
	if !client.IsGitRepo() {
		return nil, nil
	}

	ref := baseRef
	if ref == "" {
		ref = client.GetDefaultBranch()
	}

	detector := gitops.NewChangedModulesDetector(client, discovery.NewModuleIndex(nil), appCtx.WorkDir)
	return detector.DetectChangedLibraryModules(ref, libraryPaths)
}
