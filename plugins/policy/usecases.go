package policy

import (
	"context"
	"fmt"
	"io"
	"os"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/planresults"
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
	summary, err := policyusecase.Check(ctx, policyusecase.CheckRuntime{
		Config:       &cfg,
		Sources:      runtime.sources,
		WorkDir:      runtime.workDir,
		PlanSegments: runtime.planSegments,
	}, req)
	if err != nil {
		return fmt.Errorf("run policy check: %w", err)
	}

	persistPolicyArtifacts(ctx, appCtx, runtime, summary)
	return outputResult(w, format, summary, summary.HasFailures())
}

func persistPolicyArtifacts(ctx context.Context, appCtx *plugin.AppContext, runtime *policyRuntime, summary *policyengine.Summary) {
	if appCtx == nil || appCtx.Reports() == nil {
		return
	}

	artifact := reportctx.FromApp(appCtx, policyArtifactOptions(runtime))
	report, buildErr := buildPolicyReport(policyReportRequest{Summary: summary, Artifact: artifact})
	if buildErr != nil {
		log.WithError(buildErr).Warn("failed to build policy report")
		report = nil
	}
	if saveErr := appCtx.Reports().SaveResultsAndReport(ctx, pluginName, summary, report); saveErr != nil {
		log.WithError(saveErr).Warn("failed to persist policy artifacts")
	}
}

func policyArtifactOptions(runtime *policyRuntime) reportctx.Options {
	if runtime == nil {
		return reportctx.Options{}
	}
	collection, err := planresults.Scan(runtime.workDir, runtime.planSegments)
	if err != nil {
		log.WithError(err).Warn("policy: failed to fingerprint plan results")
		return reportctx.Options{}
	}
	return reportctx.Options{Collection: collection}
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
