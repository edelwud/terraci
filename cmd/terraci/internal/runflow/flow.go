// Package runflow owns the command-run lifecycle for the terraci CLI.
//
// The public command package keeps cobra wiring and command presentation; this
// package owns per-run plugin registry construction, config loading, plugin
// config decoding, preflight, AppContext construction, and pipeline
// contribution collection.
package runflow

import (
	"context"
	"fmt"
	"path/filepath"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

// RegistryFactory creates a fresh command-scoped plugin registry.
type RegistryFactory func() *registry.Registry

// LoggerInit initializes process logging for a command run.
type LoggerInit func()

// LogLevelSetter applies a parsed log level string.
type LogLevelSetter func(string) error

// Options describes a runflow instance.
type Options struct {
	RegistryFactory RegistryFactory
	InitLogger      LoggerInit
	SetLogLevel     LogLevelSetter
	Version         string
	Reports         ci.ReportStore
}

// Request describes one command pre-run lifecycle request.
type Request struct {
	CommandName string
	ConfigPath  string
	WorkDir     string
	LogLevel    string
	Verbose     bool
	Policy      CommandPolicy
}

// Flow owns command lifecycle orchestration.
type Flow struct {
	registryFactory RegistryFactory
	initLogger      LoggerInit
	setLogLevel     LogLevelSetter
	version         string
	reports         ci.ReportStore
}

// New creates a command lifecycle flow.
func New(opts Options) *Flow {
	factory := opts.RegistryFactory
	if factory == nil {
		factory = registry.New
	}
	return &Flow{
		registryFactory: factory,
		initLogger:      opts.InitLogger,
		setLogLevel:     opts.SetLogLevel,
		version:         opts.Version,
		reports:         opts.Reports,
	}
}

// Prepare executes the command pre-run lifecycle and returns the immutable
// AppContext plus the command-scoped registry/config snapshots.
func (f *Flow) Prepare(ctx context.Context, req Request) (*Prepared, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if f.initLogger != nil {
		f.initLogger()
	}
	if err := f.applyLogLevel(req); err != nil {
		return nil, err
	}

	plugins := f.registryFactory()
	if plugins == nil {
		plugins = registry.New()
	}
	if req.CommandName != "version" && f.version != "" {
		log.WithField("version", f.version).Debug("terraci")
	}

	var cfg *config.Config
	if !req.Policy.SkipConfig {
		loaded, err := loadConfig(req)
		if err != nil {
			return nil, err
		}
		cfg = loaded
		log.Debug("validating configuration")
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		if err := decodePluginConfigs(plugins, cfg); err != nil {
			return nil, err
		}
	}

	appCtx := f.buildContext(plugins, cfg, req.WorkDir)
	if !req.Policy.SkipConfig {
		if err := runPreflight(ctx, plugins, appCtx, req.Policy.SkipPreflight); err != nil {
			return nil, err
		}
		var err error
		appCtx, err = collectContributions(plugins, appCtx)
		if err != nil {
			return nil, err
		}
	}

	return newPrepared(ctx, appCtx, plugins, cfg, req.WorkDir, f.reports), nil
}

func (f *Flow) applyLogLevel(req Request) error {
	level := req.LogLevel
	if req.Verbose {
		level = "debug"
	}
	if level == "" || f.setLogLevel == nil {
		return nil
	}
	if err := f.setLogLevel(level); err != nil {
		return fmt.Errorf("invalid log level %q: %w", level, err)
	}
	return nil
}

func loadConfig(req Request) (*config.Config, error) {
	log.Debug("loading configuration")
	if req.ConfigPath != "" {
		log.WithField("file", req.ConfigPath).Debug("loading config from file")
		return config.Load(req.ConfigPath)
	}
	log.WithField("dir", req.WorkDir).Debug("loading config from directory")
	return config.LoadOrDefault(req.WorkDir)
}

func decodePluginConfigs(plugins *registry.Registry, cfg *config.Config) error {
	log.Debug("initializing plugin configurations")
	for _, p := range plugins.ConfigLoaders() {
		if _, exists := cfg.Extensions[p.ConfigKey()]; !exists {
			continue
		}
		if err := p.DecodeAndSet(func(target any) error {
			return cfg.Extension(p.ConfigKey(), target)
		}); err != nil {
			return fmt.Errorf("decode plugin config %s: %w", p.Name(), err)
		}
	}
	return nil
}

func runPreflight(ctx context.Context, plugins *registry.Registry, appCtx *plugin.AppContext, skip bool) error {
	if skip {
		log.Debug("skipping plugin preflight per command policy")
		return nil
	}
	log.Debug("running plugin preflight")
	for _, p := range plugins.PreflightsForStartup() {
		if err := p.Preflight(ctx, appCtx); err != nil {
			return fmt.Errorf("preflight plugin %s: %w", p.Name(), err)
		}
	}
	return nil
}

func collectContributions(plugins *registry.Registry, appCtx *plugin.AppContext) (*plugin.AppContext, error) {
	contributions, err := plugins.CollectContributions(appCtx)
	if err != nil {
		return nil, err
	}
	if len(contributions) == 0 {
		return appCtx, nil
	}
	return appCtx.WithPipelineContributions(contributions), nil
}

func (f *Flow) buildContext(plugins *registry.Registry, cfg *config.Config, workDir string) *plugin.AppContext {
	if f.reports == nil {
		f.reports = ci.NewFileReportStore(serviceDir(workDir, cfg))
	}
	return plugin.NewAppContext(plugin.AppContextOptions{
		Config:        cfg,
		WorkDir:       workDir,
		ServiceDir:    serviceDir(workDir, cfg),
		Version:       f.version,
		Reports:       f.reports,
		Resolver:      plugins,
		CommandLookup: plugins,
	})
}

func serviceDir(workDir string, cfg *config.Config) string {
	dir := config.DefaultServiceDir
	if cfg != nil && cfg.ServiceDir != "" {
		dir = cfg.ServiceDir
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(workDir, dir)
	}
	return dir
}
