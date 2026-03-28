package plugin

import "github.com/edelwud/terraci/pkg/config"

// AppContext is the public API available to plugins.
//
// ServiceDir is the resolved absolute path — use it for runtime file I/O.
// For pipeline artifact paths (CI templates), use Config.ServiceDir which
// preserves the original relative value from .terraci.yaml.
type AppContext struct {
	Config     *config.Config
	WorkDir    string
	ServiceDir string // resolved absolute path to project service directory
	Version    string
}
