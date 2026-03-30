// Package plugin provides the compile-time plugin system for TerraCi.
// Plugins register themselves via init() and blank imports, following the
// same pattern as database/sql drivers and Caddy modules.
package plugin

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
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
//
// Plugin state progresses through these framework stages:
//   - registered: discovered via init() + Register()
//   - configured: plugin config section was decoded successfully
//   - enabled: plugin should actively participate for this run
//   - preflighted: plugin completed cheap validation for this command
//
// IsConfigured() answers whether config exists; IsEnabled() answers whether the
// plugin should be used. Framework lifecycle should key off IsEnabled().

// Preflightable plugins run cheap validation after config is loaded, before any
// command runs. Preflight should not cache mutable command state or perform
// heavy runtime setup that can be created lazily inside plugin use-cases.
type Preflightable interface {
	Plugin
	Preflight(ctx context.Context, appCtx *AppContext) error
}

// Initializable is the legacy lifecycle hook retained for compatibility.
// Prefer Preflightable for new plugins and migrations.
type Initializable interface {
	Plugin
	Initialize(ctx context.Context, appCtx *AppContext) error
}

// Resettable plugins can reset their mutable state to zero values.
// Used by test infrastructure to isolate tests from shared plugin singletons.
type Resettable interface {
	Plugin
	Reset()
}

// --- Configuration ---

// ConfigLoader declares a config section under "plugins:" in .terraci.yaml.
// Implemented automatically by embedding BasePlugin[C].
type ConfigLoader interface {
	Plugin
	ConfigKey() string
	NewConfig() any
	DecodeAndSet(decode func(target any) error) error
	IsConfigured() bool
	IsEnabled() bool
}

// --- Commands ---

// CommandProvider adds CLI subcommands to TerraCi.
type CommandProvider interface {
	Plugin
	Commands(ctx *AppContext) []*cobra.Command
}

// --- CI Provider (split into focused interfaces) ---

// EnvDetector detects whether this plugin's CI environment is active.
type EnvDetector interface {
	Plugin
	DetectEnv() bool
}

// CIMetadata provides CI-specific metadata.
type CIMetadata interface {
	Plugin
	ProviderName() string
	PipelineID() string
	CommitSHA() string
}

// GeneratorFactory creates pipeline generators.
type GeneratorFactory interface {
	Plugin
	NewGenerator(ctx *AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator
}

// CommentFactory creates PR/MR comment services.
type CommentFactory interface {
	Plugin
	NewCommentService(ctx *AppContext) ci.CommentService
}

// CIProvider is the resolved CI provider, assembled from the focused interfaces above.
// Returned by ResolveProvider().
type CIProvider struct {
	plugin   Plugin
	metadata CIMetadata
	gen      GeneratorFactory
	comment  CommentFactory
}

// NewCIProvider constructs a CIProvider from a plugin implementing all CI interfaces.
func NewCIProvider(p Plugin, meta CIMetadata, gen GeneratorFactory, comment CommentFactory) *CIProvider {
	return &CIProvider{plugin: p, metadata: meta, gen: gen, comment: comment}
}

func (c *CIProvider) Name() string         { return c.plugin.Name() }
func (c *CIProvider) Description() string  { return c.plugin.Description() }
func (c *CIProvider) ProviderName() string { return c.metadata.ProviderName() }
func (c *CIProvider) PipelineID() string   { return c.metadata.PipelineID() }
func (c *CIProvider) CommitSHA() string    { return c.metadata.CommitSHA() }

func (c *CIProvider) NewGenerator(ctx *AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	return c.gen.NewGenerator(ctx, depGraph, modules)
}

func (c *CIProvider) NewCommentService(ctx *AppContext) ci.CommentService {
	return c.comment.NewCommentService(ctx)
}

// Plugin returns the underlying plugin instance.
func (c *CIProvider) Plugin() Plugin { return c.plugin }

// --- CLI Flag Overrides ---

// FlagOverridable plugins support direct CLI flag overrides on their config.
type FlagOverridable interface {
	Plugin
	SetPlanOnly(bool)
	SetAutoApprove(bool)
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

// --- Init Wizard ---

// InitCategory determines how an InitGroupSpec is rendered in the wizard.
type InitCategory string

const (
	// CategoryProvider groups contain CI-specific infrastructure settings (image, runner).
	// Rendered as separate groups with ShowWhen.
	CategoryProvider InitCategory = "provider"
	// CategoryPipeline groups contain pipeline behavior settings (plan_enabled, auto_approve).
	// Fields from all CategoryPipeline groups are merged into a single "Pipeline" group.
	CategoryPipeline InitCategory = "pipeline"
	// CategoryFeature groups contain optional feature toggles (cost, policy, summary).
	// Fields from all CategoryFeature groups are merged into a single "Features" group.
	CategoryFeature InitCategory = "feature"
	// CategoryDetail groups contain detail settings for enabled features (policy settings).
	// Rendered as separate groups with ShowWhen (typically gated by a feature toggle).
	CategoryDetail InitCategory = "detail"
)

// InitContributor plugins contribute fields and config to the init wizard.
type InitContributor interface {
	Plugin
	InitGroups() []*InitGroupSpec
	BuildInitConfig(state *StateMap) *InitContribution
}

// InitGroupSpec describes a group of form fields contributed by a plugin.
type InitGroupSpec struct {
	Title    string
	Category InitCategory
	Order    int
	Fields   []InitField
	ShowWhen func(*StateMap) bool
}

// InitField describes a single form field in the init wizard.
type InitField struct {
	Key         string
	Title       string
	Description string
	Type        string // "string", "bool", "select"
	Default     any
	Options     []InitOption
	Placeholder string
}

// InitOption represents a selectable option for a field.
type InitOption struct {
	Label string
	Value string
}

// InitContribution holds the config produced by a plugin's init logic.
type InitContribution struct {
	PluginKey string
	Config    map[string]any
}

// --- Pipeline Contribution ---

// PipelineContributor plugins add steps or jobs to the generated CI pipeline.
type PipelineContributor interface {
	Plugin
	PipelineContribution(ctx *AppContext) *pipeline.Contribution
}
