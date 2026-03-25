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
}

// PluginContext returns the AppContext for plugins.
// The returned context auto-refreshes from App state on Ensure().
func (a *App) PluginContext() *plugin.AppContext {
	ctx := &plugin.AppContext{
		Version: a.Version,
	}
	ctx.Refresh = func() {
		ctx.Config = a.Config
		ctx.WorkDir = a.WorkDir

		serviceDir := ".terraci"
		if a.Config != nil && a.Config.ServiceDir != "" {
			serviceDir = a.Config.ServiceDir
		}
		if !filepath.IsAbs(serviceDir) {
			ctx.ServiceDir = filepath.Join(a.WorkDir, serviceDir)
		} else {
			ctx.ServiceDir = serviceDir
		}
	}
	ctx.Refresh() // populate now if available
	return ctx
}

// InitPluginConfigs decodes plugin-specific configurations from the Plugins map
// and passes them to each ConfigProvider plugin.
func (a *App) InitPluginConfigs() error {
	for _, p := range plugin.ByCapability[plugin.ConfigProvider]() {
		if _, exists := a.Config.Plugins[p.ConfigKey()]; !exists {
			continue
		}
		cfg := p.NewConfig()
		if err := a.Config.PluginConfig(p.ConfigKey(), cfg); err != nil {
			return err
		}
		if err := p.SetConfig(cfg); err != nil {
			return err
		}
	}
	return nil
}
