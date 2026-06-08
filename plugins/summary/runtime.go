package summary

import (
	"github.com/edelwud/terraci/pkg/plugin"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal/summaryengine"
)

type summaryRuntime = summaryengine.Runtime

func newRuntime(appCtx *plugin.AppContext, cfg *summaryengine.Config) *summaryRuntime {
	normalized := cfg.Normalized()
	structure := appCtx.Config().Structure()
	segments := structure.Segments()
	return &summaryengine.Runtime{
		Config:           normalized,
		WorkDir:          appCtx.WorkDir(),
		ServiceDir:       appCtx.ServiceDir(),
		Segments:         segments,
		ProviderResolver: resolveSummaryProvider(appCtx),
		ReportStore:      appCtx.Reports(),
	}
}

// runtime returns the typed plugin runtime used by summary use-cases.
func (p *Plugin) runtime(appCtx *plugin.AppContext) *summaryRuntime {
	return newRuntime(appCtx, p.Config())
}
