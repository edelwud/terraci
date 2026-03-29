package cmd

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
)

// App holds shared state for all commands, replacing package-level globals.
type App struct {
	Config  *config.Config
	WorkDir string

	// Global flag values
	cfgFile  string
	logLevel string

	// Version info
	Version string
	Commit  string
	Date    string

	// Shared plugin context — refreshed via Ensure() after config loads
	pluginCtx *plugin.AppContext
}

// PluginContext returns the shared AppContext for plugins.
// The returned pointer is stable; call Ensure() to refresh its fields
// after Config or WorkDir change.
func (a *App) PluginContext() *plugin.AppContext {
	if a.pluginCtx == nil {
		a.pluginCtx = &plugin.AppContext{}
	}
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
	a.pluginCtx.Config = a.Config
	a.pluginCtx.WorkDir = a.WorkDir
	a.pluginCtx.ServiceDir = serviceDir
	a.pluginCtx.Version = a.Version
	if a.pluginCtx.Reports == nil {
		a.pluginCtx.Reports = plugin.NewReportRegistry()
	}
}

// InitPluginConfigs decodes plugin-specific configurations from the Plugins map
// and passes them to each ConfigLoader plugin.
func (a *App) InitPluginConfigs() error {
	for _, p := range plugin.ByCapability[plugin.ConfigLoader]() {
		if _, exists := a.Config.Plugins[p.ConfigKey()]; !exists {
			continue
		}
		if err := p.DecodeAndSet(func(target any) error {
			return a.Config.PluginConfig(p.ConfigKey(), target)
		}); err != nil {
			return err
		}
	}
	return nil
}
