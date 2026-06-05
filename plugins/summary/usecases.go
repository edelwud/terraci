package summary

import (
	"context"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal/summaryengine"
)

func resolveSummaryProvider(appCtx *plugin.AppContext) summaryengine.ProviderResolver {
	return func() (summaryengine.Provider, error) {
		provider, err := appCtx.CIResolver().ResolveCIProvider()
		if err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, nil
		}
		return summaryProvider{
			provider: provider,
			appCtx:   appCtx,
		}, nil
	}
}

type summaryProvider struct {
	provider *plugin.ResolvedCIProvider
	appCtx   *plugin.AppContext
}

func (p summaryProvider) CommitSHA() string {
	return p.provider.CommitSHA()
}

func (p summaryProvider) PipelineID() string {
	return p.provider.PipelineID()
}

func (p summaryProvider) CommentService() (ci.CommentService, bool) {
	return p.provider.NewCommentService(p.appCtx)
}

func runSummaryUseCase(ctx context.Context, appCtx *plugin.AppContext, runtime *summaryRuntime) error {
	if runtime == nil {
		runtime = newRuntime(appCtx, nil)
	}
	result, err := summaryengine.Run(ctx, *runtime, summaryengine.Request{})
	if err != nil {
		return err
	}
	printSummary(result.Snapshot.PlanResults())
	return nil
}

func (p *Plugin) runSummary(ctx context.Context, appCtx *plugin.AppContext) error {
	runtime := p.runtime(appCtx)
	return runSummaryUseCase(ctx, appCtx, runtime)
}
