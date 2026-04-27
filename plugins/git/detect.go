package git

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	gitclient "github.com/edelwud/terraci/plugins/git/internal"
)

// DetectChangedModules returns modules changed since the given base ref.
func (p *Plugin) DetectChangedModules(_ context.Context, workDir, baseRef string, moduleIndex *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	client := gitclient.NewClient(workDir)
	if !client.IsGitRepo() {
		return nil, nil, fmt.Errorf("not a git repository: %s", workDir)
	}

	ref := p.resolveRef(baseRef, client)
	detector := gitclient.NewChangedModulesDetector(client, moduleIndex, workDir)
	return detector.DetectChangedModulesVerbose(ref)
}

// DetectChangedLibraries returns library paths changed since the given base ref.
func (p *Plugin) DetectChangedLibraries(_ context.Context, workDir, baseRef string, libraryPaths []string) ([]string, error) {
	client := gitclient.NewClient(workDir)
	if !client.IsGitRepo() {
		return nil, nil
	}

	ref := p.resolveRef(baseRef, client)
	detector := gitclient.NewChangedModulesDetector(client, discovery.NewModuleIndex(nil), workDir)
	return detector.DetectChangedLibraryModules(ref, libraryPaths)
}
