package policy

import (
	"context"
	"fmt"
	"io"
	"os"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/cliout"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
	policyusecase "github.com/edelwud/terraci/plugins/policy/internal/usecase"
)

func runPullPoliciesUseCase(ctx context.Context, runtime *policyRuntime, req policyengine.PullRequest) error {
	result, err := policyusecase.Pull(ctx, runtime.sources, req)
	if err != nil {
		return fmt.Errorf("failed to materialize policies: %w", err)
	}

	log.WithField("count", len(result.PolicyDirs)).Info("policy sources materialized")
	log.WithField("cache", result.CacheDir).Info("policies cached")
	return nil
}

type policyCheckResult struct {
	Summary     *policyengine.Summary
	PlanResults *ci.PlanResultCollection
}

func runPolicyCheckUseCase(ctx context.Context, runtime *policyRuntime, req policyengine.CheckRequest) (*policyCheckResult, error) {
	cfg := runtime.config
	checkResult, err := policyusecase.Check(ctx, policyusecase.CheckRuntime{
		Config:       &cfg,
		Sources:      runtime.sources,
		WorkDir:      runtime.workDir,
		PlanSegments: runtime.planSegments,
	}, req)
	if err != nil {
		return nil, fmt.Errorf("run policy check: %w", err)
	}

	return &policyCheckResult{
		Summary:     checkResult.Summary,
		PlanResults: checkResult.PlanResults,
	}, nil
}

func persistPolicyArtifacts(ctx context.Context, appCtx *plugin.AppContext, summary *policyengine.Summary, collection *ci.PlanResultCollection) {
	if appCtx == nil || appCtx.Reports() == nil {
		return
	}

	publication, err := ci.NewArtifactPublication(ci.ArtifactPublicationOptions{
		Producer: pluginName,
		Results:  ci.RawResults(summary),
		BuildReport: func() (*ci.Report, error) {
			run, err := plugin.NewArtifactRun(appCtx, plugin.ArtifactRunOptions{
				Producer:   pluginName,
				Collection: collection,
			})
			if err != nil {
				return nil, fmt.Errorf("artifact run: %w", err)
			}
			return buildPolicyReport(policyReportRequest{Summary: summary, Run: run})
		},
	})
	if err != nil {
		log.WithError(err).Warn("failed to persist policy artifacts")
		return
	}
	if saveErr := appCtx.Reports().PublishArtifacts(ctx, publication); saveErr != nil {
		log.WithError(saveErr).Warn("failed to persist policy artifacts")
	}
}

func (p *Plugin) runPull(ctx context.Context, appCtx *plugin.AppContext, cacheDir string) error {
	runtime, err := p.runtime(ctx, appCtx)
	if err != nil {
		return err
	}

	return runPullPoliciesUseCase(ctx, runtime, policyengine.PullRequest{CacheDir: cacheDir})
}

func (p *Plugin) runCheck(ctx context.Context, appCtx *plugin.AppContext, modulePath, format string, w io.Writer) error {
	outputFmt, err := cliout.ParseFormat(format)
	if err != nil {
		return err
	}

	runtime, err := p.runtime(ctx, appCtx)
	if err != nil {
		return err
	}

	if w == nil {
		w = os.Stdout
	}
	result, err := runPolicyCheckUseCase(ctx, runtime, policyengine.CheckRequest{ModulePath: modulePath})
	if err != nil {
		return err
	}
	persistPolicyArtifacts(ctx, appCtx, result.Summary, result.PlanResults)
	return outputResult(w, outputFmt, result.Summary, result.Summary.HasFailures())
}
