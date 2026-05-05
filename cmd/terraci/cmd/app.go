package cmd

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

// App holds shared state for all commands, replacing package-level globals.
//
// Plugins is the per-command plugin registry, rebuilt fresh in PreRunE for
// every cobra command run. Code that runs *before* PreRunE (root command
// registration in NewRootCmd) holds an initial registry created in newApp;
// after PreRunE that initial registry is replaced by a freshly built one
// and the AppContext is constructed once and attached to cmd.Context().
type App struct {
	Config  *config.Config
	WorkDir string
	Plugins *registry.Registry

	// Global flag values
	cfgFile  string
	logLevel string

	// Version info
	Version string
	Commit  string
	Date    string

	// reports is the long-lived in-process registry. AppContexts created
	// for each command share this registry so mid-command report exchange
	// keeps working across cobra command boundaries.
	reports *plugin.ReportRegistry
}

func newApp(version, commit, date string) *App {
	return &App{
		Version: version,
		Commit:  commit,
		Date:    date,
		Plugins: registry.New(),
		reports: plugin.NewReportRegistry(),
	}
}

// BuildContext constructs a fresh immutable AppContext bound to the current
// App state. Called once per command run from PersistentPreRunE; the
// returned context is attached to cmd.Context() so plugins can retrieve it
// via plugin.FromContext.
func (a *App) BuildContext() *plugin.AppContext {
	if a.Plugins == nil {
		a.Plugins = registry.New()
	}
	if a.reports == nil {
		a.reports = plugin.NewReportRegistry()
	}
	return plugin.NewAppContext(plugin.AppContextOptions{
		Config:     a.Config,
		WorkDir:    a.WorkDir,
		ServiceDir: a.serviceDir(),
		Version:    a.Version,
		Reports:    a.reports,
		Resolver:   a.Plugins,
	})
}

// ResetPluginsForCommand swaps in a fresh per-command plugin registry. Use
// from PersistentPreRunE before BuildContext.
func (a *App) ResetPluginsForCommand() {
	a.Plugins = registry.New()
}

func (a *App) serviceDir() string {
	dir := ".terraci"
	if a.Config != nil && a.Config.ServiceDir != "" {
		dir = a.Config.ServiceDir
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(a.WorkDir, dir)
	}
	return dir
}

// InitPluginConfigs decodes plugin-specific configurations from the Plugins
// registry and passes them to each ConfigLoader plugin.
func (a *App) InitPluginConfigs() error {
	for _, p := range registry.ByCapabilityFrom[plugin.ConfigLoader](a.Plugins) {
		if _, exists := a.Config.Extensions[p.ConfigKey()]; !exists {
			continue
		}
		if err := p.DecodeAndSet(func(target any) error {
			return a.Config.Extension(p.ConfigKey(), target)
		}); err != nil {
			return err
		}
	}
	return nil
}
