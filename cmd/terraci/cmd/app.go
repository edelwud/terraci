package cmd

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

// App holds shared state for all commands, replacing package-level globals.
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

	// Shared plugin context pointer used by commands registered before command execution.
	pluginCtx *plugin.AppContext
}

// BeginCommand creates a fresh plugin instance set and context for one command execution.
func (a *App) BeginCommand() {
	a.Plugins = registry.New()
	if a.pluginCtx != nil {
		a.pluginCtx.BeginCommand(a.Plugins)
	}
}

// PluginContext returns the shared AppContext pointer for plugins.
// The pointer is stable for cobra command closures; its contents are rebound to
// the current command's registry/config during PersistentPreRunE.
func (a *App) PluginContext() *plugin.AppContext {
	if a.Plugins == nil {
		a.Plugins = registry.New()
	}
	if a.pluginCtx == nil {
		a.pluginCtx = plugin.NewAppContext(nil, "", "", "", nil, a.Plugins)
	}
	a.pluginCtx.SetResolver(a.Plugins)
	a.ensurePluginContext()
	return a.pluginCtx
}

// ensurePluginContext updates the shared AppContext fields from current App state.
func (a *App) ensurePluginContext() {
	if a.pluginCtx.IsFrozen() {
		return
	}
	serviceDir := ".terraci"
	if a.Config != nil && a.Config.ServiceDir != "" {
		serviceDir = a.Config.ServiceDir
	}
	if !filepath.IsAbs(serviceDir) {
		serviceDir = filepath.Join(a.WorkDir, serviceDir)
	}
	a.pluginCtx.Update(a.Config, a.WorkDir, serviceDir, a.Version)
}

// InitPluginConfigs decodes plugin-specific configurations from the Plugins map
// and passes them to each ConfigLoader plugin.
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
