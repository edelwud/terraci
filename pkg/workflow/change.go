package workflow

import (
	"context"

	"github.com/edelwud/terraci/pkg/discovery"
)

// ChangeDetectionRequest describes one VCS diff request. Implementations
// should derive all change dimensions from a single underlying diff.
type ChangeDetectionRequest struct {
	WorkDir      string
	BaseRef      string
	ModuleIndex  *discovery.ModuleIndex
	LibraryPaths []string
}

// ChangeDetectionResult contains changed files and their TerraCi projections.
type ChangeDetectionResult struct {
	Modules      []*discovery.Module
	Files        []string
	LibraryPaths []string
}

// ChangeDetector detects changed modules from git or another VCS.
type ChangeDetector interface {
	DetectChanges(ctx context.Context, req ChangeDetectionRequest) (*ChangeDetectionResult, error)
}
