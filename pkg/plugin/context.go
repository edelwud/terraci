package plugin

import (
	"sync"

	"github.com/edelwud/terraci/pkg/config"
)

// AppContext is the public API available to plugins.
//
// ServiceDir is the resolved absolute path — use it for runtime file I/O.
// For pipeline artifact paths (CI templates), use Config.ServiceDir which
// preserves the original relative value from .terraci.yaml.
//
// After initialization, call Freeze() to prevent the framework from refreshing
// this shared context from App state again during the current command run.
//
// The Config returned by Config() is shared and should be treated as read-only
// by plugins. Mutate a deep copy if a plugin needs to derive a configuration.
//
// AppContext is safe for concurrent use: all accessors take an RWMutex,
// allowing background goroutines (cost engine, registry clients, runners)
// to read state while the framework rebinds it across command runs.
type AppContext struct {
	mu sync.RWMutex // protects all fields below

	config     *config.Config
	workDir    string
	serviceDir string // resolved absolute path to project service directory
	version    string
	reports    *ReportRegistry
	resolver   Resolver

	frozen bool
}

// AppContextOptions describes how to construct an AppContext. All fields are
// optional except WorkDir / ServiceDir, which are required for runtime I/O
// resolution. A zero AppContextOptions is sometimes useful as a sentinel; in
// production code prefer setting at least Config, WorkDir, and ServiceDir.
type AppContextOptions struct {
	// Config is the loaded TerraCi configuration. Treated as read-only.
	Config *config.Config
	// WorkDir is the project working directory for the current command.
	WorkDir string
	// ServiceDir is the resolved absolute service directory path.
	ServiceDir string
	// Version is the current TerraCi version string.
	Version string
	// Reports is the shared in-process report registry. Defaults to a fresh
	// empty registry if nil.
	Reports *ReportRegistry
	// Resolver is the per-run plugin resolver. Defaults to a no-op resolver
	// that returns nothing if nil — plugins may always call ctx.Resolver().
	Resolver Resolver
}

// NewAppContext creates a framework-managed plugin context.
func NewAppContext(opts AppContextOptions) *AppContext {
	reports := opts.Reports
	if reports == nil {
		reports = NewReportRegistry()
	}
	resolver := opts.Resolver
	if resolver == nil {
		resolver = noopResolver{}
	}

	ctx := &AppContext{reports: reports, resolver: resolver}
	ctx.Update(opts.Config, opts.WorkDir, opts.ServiceDir, opts.Version)
	return ctx
}

// Config returns the loaded TerraCi configuration. The returned pointer is
// shared with the framework and must not be mutated by plugins.
func (ctx *AppContext) Config() *config.Config {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.config
}

// WorkDir returns the working directory for the current command.
func (ctx *AppContext) WorkDir() string {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.workDir
}

// ServiceDir returns the resolved absolute service directory path.
func (ctx *AppContext) ServiceDir() string {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.serviceDir
}

// Version returns the current TerraCi version string.
func (ctx *AppContext) Version() string {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.version
}

// Reports returns the shared in-process report registry.
func (ctx *AppContext) Reports() *ReportRegistry {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.reports
}

// Resolver returns the per-run plugin resolver bound to this context.
// Always non-nil — when no resolver is configured, returns a no-op resolver
// whose Resolve* methods return errors and whose lookups return nothing.
func (ctx *AppContext) Resolver() Resolver {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.resolver
}
