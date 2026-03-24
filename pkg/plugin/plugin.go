// Package plugin provides the compile-time plugin system for TerraCi.
// Plugins register themselves via init() and blank imports, following the
// same pattern as database/sql drivers and Caddy modules.
package plugin

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
)

// Plugin is the core interface every plugin must implement.
type Plugin interface {
	// Name returns a unique identifier (e.g., "gitlab", "cost", "slack").
	Name() string
	// Description returns a human-readable description.
	Description() string
}

// --- Lifecycle interfaces ---

// Initializable plugins set up resources after config is loaded, before any command runs.
type Initializable interface {
	Plugin
	Initialize(ctx context.Context, appCtx *AppContext) error
}

// Finalizable plugins clean up resources after the command completes.
type Finalizable interface {
	Plugin
	Finalize(ctx context.Context) error
}

// --- Configuration ---

// ConfigProvider declares a config section under "plugins:" in .terraci.yaml.
type ConfigProvider interface {
	Plugin
	ConfigKey() string
	NewConfig() any
	SetConfig(cfg any) error
}

// --- Commands ---

// CommandProvider adds CLI subcommands to TerraCi.
type CommandProvider interface {
	Plugin
	Commands(ctx *AppContext) []*cobra.Command
}

// --- CI Provider ---

// GeneratorProvider supplies a pipeline generator and comment service for a CI provider.
type GeneratorProvider interface {
	Plugin
	ProviderName() string
	DetectEnv() bool
	NewGenerator(ctx *AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator
	NewCommentService(ctx *AppContext) ci.CommentService
}

// --- Summary ---

// SummaryContributor plugins enrich plan results during `terraci summary`.
// Called in registration order before comment rendering.
type SummaryContributor interface {
	Plugin
	ContributeToSummary(ctx context.Context, appCtx *AppContext, execCtx *ExecutionContext) error
}

// --- Version ---

// VersionProvider plugins contribute version info to `terraci version`.
type VersionProvider interface {
	Plugin
	VersionInfo() map[string]string
}

// --- Change Detection ---

// ChangeDetectionProvider detects changed modules from git (or other VCS).
type ChangeDetectionProvider interface {
	Plugin
	DetectChangedModules(ctx context.Context, appCtx *AppContext, baseRef string, moduleIndex *discovery.ModuleIndex) (changed []*discovery.Module, changedFiles []string, err error)
	DetectChangedLibraries(ctx context.Context, appCtx *AppContext, baseRef string, libraryPaths []string) ([]string, error)
}

// --- Filtering ---

// FilterProvider registers custom module filters.
type FilterProvider interface {
	Plugin
	Filters(ctx *AppContext) []filter.ModuleFilter
}

// --- Workflow Hooks ---

// WorkflowHookProvider injects behavior at workflow stages.
type WorkflowHookProvider interface {
	Plugin
	WorkflowHooks() []WorkflowHook
}
