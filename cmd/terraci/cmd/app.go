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
func (a *App) PluginContext() *plugin.AppContext {
	serviceDir := ".terraci"
	if a.Config != nil && a.Config.ServiceDir != "" {
		serviceDir = a.Config.ServiceDir
	}
	if !filepath.IsAbs(serviceDir) {
		serviceDir = filepath.Join(a.WorkDir, serviceDir)
	}
	return &plugin.AppContext{
		Config:     a.Config,
		WorkDir:    a.WorkDir,
		ServiceDir: serviceDir,
		Version:    a.Version,
	}
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
