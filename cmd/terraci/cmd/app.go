package cmd

import (
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
func (a *App) PluginContext() *plugin.AppContext {
	return &plugin.AppContext{
		Config:  a.Config,
		WorkDir: a.WorkDir,
		Version: a.Version,
	}
}

// InitPluginConfigs decodes plugin-specific configurations from the Plugins map
// and passes them to each ConfigProvider plugin.
func (a *App) InitPluginConfigs() error {
	for _, p := range plugin.ByCapability[plugin.ConfigProvider]() {
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
