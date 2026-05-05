package git

import (
	"context"
	"errors"
	"fmt"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/discovery"
	gitclient "github.com/edelwud/terraci/plugins/git/internal"
)

// DetectChangedModules returns modules changed since the given base ref.
func (p *Plugin) DetectChangedModules(_ context.Context, workDir, baseRef string, moduleIndex *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	client := gitclient.NewClient(workDir)
	if !client.IsGitRepo() {
		return nil, nil, fmt.Errorf("not a git repository: %s", workDir)
	}

	if err := p.maybeAutoUnshallow(client); err != nil {
		return nil, nil, err
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

	if err := p.maybeAutoUnshallow(client); err != nil {
		return nil, err
	}

	ref := p.resolveRef(baseRef, client)
	detector := gitclient.NewChangedModulesDetector(client, discovery.NewModuleIndex(nil), workDir)
	return detector.DetectChangedLibraryModules(ref, libraryPaths)
}

// maybeAutoUnshallow deepens a shallow clone in-place when the user has
// opted in via extensions.git.auto_unshallow. Returns nil when the repo is
// already complete or auto_unshallow is disabled. Surfaces unshallow errors
// alongside ErrShallowRepository so callers' existing error-matching logic
// keeps working when a failed auto-unshallow blocks change detection.
func (p *Plugin) maybeAutoUnshallow(client *gitclient.Client) error {
	cfg := p.Config()
	if cfg == nil || !cfg.AutoUnshallow {
		return nil
	}
	shallow, err := client.IsShallow()
	if err != nil {
		return fmt.Errorf("inspect shallow state: %w", err)
	}
	if !shallow {
		return nil
	}

	log.Info("git: shallow clone detected, deepening history via go-git fetch (auto_unshallow enabled)")
	if err := client.Unshallow(); err != nil {
		return errors.Join(gitclient.ErrShallowRepository, err)
	}
	return nil
}
