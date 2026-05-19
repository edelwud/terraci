package policy

import (
	"context"
	"fmt"
	"io"
	"os"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/internal/reportctx"
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

func runPolicyCheckUseCase(ctx context.Context, appCtx *plugin.AppContext, runtime *policyRuntime, req policyengine.CheckRequest, format outputFormat, w io.Writer) error {
	cfg := runtime.config
	checkResult, err := policyusecase.Check(ctx, policyusecase.CheckRuntime{
		Config:       &cfg,
		Sources:      runtime.sources,
		WorkDir:      runtime.workDir,
		PlanSegments: runtime.planSegments,
	}, req)
	if err != nil {
		return fmt.Errorf("run policy check: %w", err)
	}

	summary := checkResult.Summary
	persistPolicyArtifacts(ctx, appCtx, summary, checkResult.PlanResults)
	return outputResult(w, format, summary, summary.HasFailures())
}

func persistPolicyArtifacts(ctx context.Context, appCtx *plugin.AppContext, summary *policyengine.Summary, collection *ci.PlanResultCollection) {
	if appCtx == nil || appCtx.Reports() == nil {
		return
	}

	run, runErr := reportctx.NewRun(appCtx, reportctx.Options{
		Producer:   pluginName,
		Collection: collection,
	})
	if runErr != nil {
		log.WithError(runErr).Warn("failed to build policy artifact context")
	}
	report, buildErr := buildPolicyReport(policyReportRequest{Summary: summary, Run: run})
	if buildErr != nil {
		log.WithError(buildErr).Warn("failed to build policy report")
		report = nil
	}
	if runErr != nil {
		report = nil
	}
	if saveErr := appCtx.Reports().ReplaceResultsAndReport(ctx, pluginName, summary, report); saveErr != nil {
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
	outputFmt, err := parseOutputFormat(format)
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
	return runPolicyCheckUseCase(ctx, appCtx, runtime, policyengine.CheckRequest{ModulePath: modulePath}, outputFmt, w)
}
