package plugin

import "github.com/edelwud/terraci/pkg/config"

// AppContext is the public API available to plugins.
//
// ServiceDir is the resolved absolute path — use it for runtime file I/O.
// For pipeline artifact paths (CI templates), use Config.ServiceDir which
// preserves the original relative value from .terraci.yaml.
//
// After initialization, call Freeze() to prevent further mutations.
type AppContext struct {
	Config     *config.Config
	WorkDir    string
	ServiceDir string // resolved absolute path to project service directory
	Version    string
	Reports    *ReportRegistry

	frozen bool
}

// Freeze marks the context as immutable. Subsequent calls to Update will panic.
func (ctx *AppContext) Freeze() {
	ctx.frozen = true
}

// IsFrozen returns whether the context has been frozen.
func (ctx *AppContext) IsFrozen() bool {
	return ctx.frozen
}
