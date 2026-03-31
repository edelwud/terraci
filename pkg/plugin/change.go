package plugin

import (
	"context"

	"github.com/edelwud/terraci/pkg/discovery"
)

// ChangeDetectionProvider detects changed modules from git (or other VCS).
type ChangeDetectionProvider interface {
	Plugin
	DetectChangedModules(ctx context.Context, appCtx *AppContext, baseRef string, moduleIndex *discovery.ModuleIndex) (changed []*discovery.Module, changedFiles []string, err error)
	DetectChangedLibraries(ctx context.Context, appCtx *AppContext, baseRef string, libraryPaths []string) ([]string, error)
}
