package git

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/plugin"
	gitclient "github.com/edelwud/terraci/plugins/git/internal"
)

// DetectChangedModules returns modules changed since the given base ref.
func (p *Plugin) DetectChangedModules(_ context.Context, appCtx *plugin.AppContext, baseRef string, moduleIndex *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	client := p.getClient()
	if client == nil {
		return nil, nil, fmt.Errorf("not a git repository: %s", appCtx.WorkDir)
	}

	ref := p.resolveRef(baseRef)
	detector := gitclient.NewChangedModulesDetector(client, moduleIndex, appCtx.WorkDir)
	return detector.DetectChangedModulesVerbose(ref)
}

// DetectChangedLibraries returns library paths changed since the given base ref.
func (p *Plugin) DetectChangedLibraries(_ context.Context, appCtx *plugin.AppContext, baseRef string, libraryPaths []string) ([]string, error) {
	client := p.getClient()
	if client == nil {
		return nil, nil
	}

	ref := p.resolveRef(baseRef)
	detector := gitclient.NewChangedModulesDetector(client, discovery.NewModuleIndex(nil), appCtx.WorkDir)
	return detector.DetectChangedLibraryModules(ref, libraryPaths)
}
