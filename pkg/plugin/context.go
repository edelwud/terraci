package plugin

import (
	"context"

	"github.com/edelwud/terraci/pkg/config"
)

// AppContext is the public API available to plugins. It is immutable —
// constructed once per command run by the framework, then read-only.
//
// ServiceDir is the resolved absolute path; use it for runtime file I/O.
// For pipeline artifact paths (CI templates), use Config.ServiceDir which
// preserves the original relative value from .terraci.yaml.
//
// The Config returned by Config() is shared and must be treated as
// read-only by plugins. Mutate a deep copy if a plugin needs to derive a
// configuration.
//
// AppContext is safe for concurrent reads from any goroutine because all
// fields are written exactly once at construction.
type AppContext struct {
	config     *config.Config
	workDir    string
	serviceDir string
	version    string
	reports    *ReportRegistry
	resolver   Resolver
}

// AppContextOptions describes how to construct an AppContext.
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
	// Resolver is the per-run plugin resolver. Defaults to NoopResolver{}
	// when nil — plugins may always call ctx.Resolver() without nil-checks.
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
		resolver = NoopResolver{}
	}
	return &AppContext{
		config:     opts.Config,
		workDir:    opts.WorkDir,
		serviceDir: opts.ServiceDir,
		version:    opts.Version,
		reports:    reports,
		resolver:   resolver,
	}
}

// Config returns the loaded TerraCi configuration. The returned pointer is
// shared with the framework and must not be mutated by plugins.
func (ctx *AppContext) Config() *config.Config { return ctx.config }

// WorkDir returns the working directory for the current command.
func (ctx *AppContext) WorkDir() string { return ctx.workDir }

// ServiceDir returns the resolved absolute service directory path.
func (ctx *AppContext) ServiceDir() string { return ctx.serviceDir }

// Version returns the current TerraCi version string.
func (ctx *AppContext) Version() string { return ctx.version }

// Reports returns the shared in-process report registry.
func (ctx *AppContext) Reports() *ReportRegistry { return ctx.reports }

// Resolver returns the per-run plugin resolver. Always non-nil.
func (ctx *AppContext) Resolver() Resolver { return ctx.resolver }

// appContextKey is the unexported key under which AppContext is carried in
// context.Context. Plugins access the value via FromContext.
type appContextKey struct{}

// WithContext returns a child context.Context carrying appCtx. Used by the
// framework to attach the per-run AppContext to the cobra command context
// before RunE fires.
func WithContext(parent context.Context, appCtx *AppContext) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, appContextKey{}, appCtx)
}

// FromContext retrieves the AppContext attached to ctx, or nil if none is
// bound. Plugins use this inside cobra RunE callbacks:
//
//	appCtx := plugin.FromContext(cmd.Context())
func FromContext(ctx context.Context) *AppContext {
	if ctx == nil {
		return nil
	}
	v, ok := ctx.Value(appContextKey{}).(*AppContext)
	if !ok {
		return nil
	}
	return v
}
