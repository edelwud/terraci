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
	workDir := appCtx.WorkDir()
	client := p.getClient()
	if client == nil {
		return nil, nil, fmt.Errorf("not a git repository: %s", workDir)
	}

	ref := p.resolveRef(baseRef)
	detector := gitclient.NewChangedModulesDetector(client, moduleIndex, workDir)
	return detector.DetectChangedModulesVerbose(ref)
}

// DetectChangedLibraries returns library paths changed since the given base ref.
func (p *Plugin) DetectChangedLibraries(_ context.Context, appCtx *plugin.AppContext, baseRef string, libraryPaths []string) ([]string, error) {
	workDir := appCtx.WorkDir()
	client := p.getClient()
	if client == nil {
		return nil, nil
	}

	ref := p.resolveRef(baseRef)
	detector := gitclient.NewChangedModulesDetector(client, discovery.NewModuleIndex(nil), workDir)
	return detector.DetectChangedLibraryModules(ref, libraryPaths)
}
