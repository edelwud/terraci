package plugin

import (
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
)

// AppContext is the public API available to plugins. It is immutable —
// constructed once per command run by the framework, then read-only.
//
// ServiceDir is the resolved absolute path; use it for runtime file I/O.
// For pipeline artifact paths (CI templates), use Config().ServiceDir() which
// preserves the original relative value from .terraci.yaml.
//
// Config returns immutable TerraCi config. Production code should consume
// accessors and build a new config value for modified fixtures.
//
// AppContext is safe for concurrent reads from any goroutine because all
// fields are written exactly once at construction.
type AppContext struct {
	config     config.Config
	workDir    string
	serviceDir string
	version    string
	reports    ci.ReportStore
	resolvers  ResolverSet
}

// AppContextOptions describes how to construct an AppContext.
type AppContextOptions struct {
	// Config is the loaded immutable TerraCi configuration.
	Config config.Config
	// WorkDir is the project working directory for the current command.
	WorkDir string
	// ServiceDir is the resolved absolute service directory path.
	ServiceDir string
	// Version is the current TerraCi version string.
	Version string
	// Reports is the shared report store. Defaults to a file-backed store when
	// ServiceDir is set, otherwise to an in-process memory store.
	Reports ci.ReportStore
	// Resolvers is the per-run narrow resolver bundle. Missing capabilities
	// default to no-op resolvers. Plugins consume it through AppContext's
	// narrow resolver accessors.
	Resolvers ResolverSet
}

// NewAppContext creates a framework-managed plugin context.
func NewAppContext(opts AppContextOptions) *AppContext {
	reports := opts.Reports
	if reports == nil {
		if opts.ServiceDir != "" {
			reports = ci.NewFileReportStore(opts.ServiceDir)
		} else {
			reports = ci.NewMemoryReportStore()
		}
	}
	return &AppContext{
		config:     opts.Config,
		workDir:    opts.WorkDir,
		serviceDir: opts.ServiceDir,
		version:    opts.Version,
		reports:    reports,
		resolvers:  opts.Resolvers,
	}
}

// Config returns the loaded TerraCi configuration snapshot.
func (ctx *AppContext) Config() config.Config { return ctx.config }

// WorkDir returns the working directory for the current command.
func (ctx *AppContext) WorkDir() string { return ctx.workDir }

// ServiceDir returns the resolved absolute service directory path.
func (ctx *AppContext) ServiceDir() string { return ctx.serviceDir }

// Version returns the current TerraCi version string.
func (ctx *AppContext) Version() string { return ctx.version }

// Reports returns the shared report store.
func (ctx *AppContext) Reports() ci.ReportStore { return ctx.reports }

// CIResolver returns the active CI provider resolver. Always non-nil.
func (ctx *AppContext) CIResolver() CIResolver { return ctx.resolvers.CIResolver() }

// ChangeDetectorResolver returns the active change detector resolver. Always non-nil.
func (ctx *AppContext) ChangeDetectorResolver() ChangeDetectorResolver {
	return ctx.resolvers.ChangeDetectorResolver()
}

// KVCacheResolver returns the named KV cache backend resolver. Always non-nil.
func (ctx *AppContext) KVCacheResolver() KVCacheResolver { return ctx.resolvers.KVCacheResolver() }

// BlobStoreResolver returns the named blob store backend resolver. Always non-nil.
func (ctx *AppContext) BlobStoreResolver() BlobStoreResolver {
	return ctx.resolvers.BlobStoreResolver()
}
