package summary

import (
	"context"

	"github.com/edelwud/terraci/pkg/plugin"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal/summaryengine"
)

type summaryRuntime = summaryengine.Runtime

func newRuntime(appCtx *plugin.AppContext, cfg *summaryengine.Config) *summaryRuntime {
	normalized := cfg.Normalized()
	structure := appCtx.Config().Structure()
	segments := append([]string(nil), structure.Segments...)
	return &summaryengine.Runtime{
		Config:           normalized,
		WorkDir:          appCtx.WorkDir(),
		ServiceDir:       appCtx.ServiceDir(),
		Segments:         segments,
		ProviderResolver: resolveSummaryProvider(appCtx),
		ReportStore:      appCtx.Reports(),
	}
}

func (p *Plugin) Runtime(_ context.Context, appCtx *plugin.AppContext) (any, error) {
	return newRuntime(appCtx, p.Config()), nil
}

// runtime returns the typed plugin runtime used by summary use-cases.
func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext) (*summaryRuntime, error) {
	return plugin.BuildRuntime[*summaryRuntime](ctx, p, appCtx)
}
