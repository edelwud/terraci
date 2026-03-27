package plugin

import "github.com/edelwud/terraci/pkg/config"

// AppContext is the public API available to plugins.
type AppContext struct {
	Config     *config.Config
	WorkDir    string
	ServiceDir string // resolved absolute path to project service directory
	Version    string
}
