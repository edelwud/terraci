package git

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/git/internal/gitclient"
)

// DetectChanges returns changed files plus module/library projections.
func (p *Plugin) DetectChanges(ctx context.Context, req workflow.ChangeDetectionRequest) (*workflow.ChangeDetectionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	workDir := req.WorkDir
	if workDir == "" {
		workDir = "."
	}
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve workdir %q: %w", workDir, err)
	}

	client := gitclient.NewClient(absWorkDir)
	if !client.IsGitRepo() {
		return nil, fmt.Errorf("not a git repository: %s", absWorkDir)
	}

	moduleIndex := req.ModuleIndex
	if moduleIndex == nil {
		moduleIndex = discovery.NewModuleIndex(nil)
	}

	ref := client.ResolveBaseRef(req.BaseRef)
	detector := gitclient.NewChangedModulesDetector(client, moduleIndex, absWorkDir)
	modules, files, libraries, err := detector.DetectChanges(ref, req.LibraryPaths)
	if err != nil {
		return nil, fmt.Errorf("git diff against %q: %w", ref, err)
	}

	return &workflow.ChangeDetectionResult{
		Modules:      modules,
		Files:        files,
		LibraryPaths: libraries,
	}, nil
}
