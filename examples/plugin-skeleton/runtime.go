package skeleton

import (
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
)

// Runtime is the immutable dependency bundle used by command usecases.
type Runtime struct {
	Config     *Config
	WorkDir    string
	ServiceDir string
	Reports    ci.ReportStore
}

// NewRuntime builds the command runtime from the framework AppContext.
func NewRuntime(appCtx *plugin.AppContext, cfg *Config) Runtime {
	if appCtx == nil {
		return Runtime{Config: cfg.Clone()}
	}
	return Runtime{
		Config:     cfg.Clone(),
		WorkDir:    appCtx.WorkDir(),
		ServiceDir: appCtx.ServiceDir(),
		Reports:    appCtx.Reports(),
	}
}
