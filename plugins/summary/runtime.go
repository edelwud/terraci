package summary

import (
	"context"

	"github.com/edelwud/terraci/pkg/plugin"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal"
)

type summaryRuntime struct {
	config          *summaryengine.Config
	resolveProvider func() (summaryProvider, error)
}

func newRuntime(appCtx *plugin.AppContext, cfg *summaryengine.Config) *summaryRuntime {
	if cfg == nil {
		cfg = &summaryengine.Config{}
	}
	return &summaryRuntime{
		config:          cfg,
		resolveProvider: resolveSummaryProvider(appCtx),
	}
}

func (p *Plugin) Runtime(_ context.Context, appCtx *plugin.AppContext) (any, error) {
	return newRuntime(appCtx, p.Config()), nil
}

// runtime returns the typed plugin runtime used by summary use-cases.
func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext) (*summaryRuntime, error) {
	return plugin.BuildRuntime[*summaryRuntime](ctx, p, appCtx)
}
