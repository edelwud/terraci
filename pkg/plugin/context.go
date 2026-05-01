package plugin

import "github.com/edelwud/terraci/pkg/config"

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
type AppContext struct {
	config     *config.Config
	workDir    string
	serviceDir string // resolved absolute path to project service directory
	version    string
	reports    *ReportRegistry
	resolver   Resolver

	frozen bool
}

// NewAppContext creates a framework-managed plugin context.
func NewAppContext(cfg *config.Config, workDir, serviceDir, version string, reports *ReportRegistry, resolver ...Resolver) *AppContext {
	if reports == nil {
		reports = NewReportRegistry()
	}
	ctx := &AppContext{reports: reports}
	if len(resolver) > 0 {
		ctx.resolver = resolver[0]
	}
	ctx.Update(cfg, workDir, serviceDir, version)
	return ctx
}

// Config returns the loaded TerraCi configuration. The returned pointer is
// shared with the framework and must not be mutated by plugins.
func (ctx *AppContext) Config() *config.Config {
	return ctx.config
}

// WorkDir returns the working directory for the current command.
func (ctx *AppContext) WorkDir() string {
	return ctx.workDir
}

// ServiceDir returns the resolved absolute service directory path.
func (ctx *AppContext) ServiceDir() string {
	return ctx.serviceDir
}

// Version returns the current TerraCi version string.
func (ctx *AppContext) Version() string {
	return ctx.version
}

// Reports returns the shared in-process report registry.
func (ctx *AppContext) Reports() *ReportRegistry {
	return ctx.reports
}

// Resolver returns the per-run plugin resolver bound to this context.
func (ctx *AppContext) Resolver() Resolver {
	return ctx.resolver
}
