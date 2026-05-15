package policy

import (
	"context"
	"fmt"
	"io"
	"os"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
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

func runPolicyCheckUseCase(ctx context.Context, runtime *policyRuntime, req policyengine.CheckRequest, format outputFormat, w io.Writer) error {
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

	persistPolicyArtifacts(runtime.serviceDir, summary)
	return outputResult(w, format, summary, summary.HasFailures())
}

func persistPolicyArtifacts(serviceDir string, summary *policyengine.Summary) {
	if serviceDir == "" {
		return
	}

	report, buildErr := buildPolicyReport(summary)
	if buildErr != nil {
		log.WithError(buildErr).Warn("failed to build policy report")
		report = nil
	}
	if saveErr := ci.SaveResultsAndReport(serviceDir, resultsFile, summary, report); saveErr != nil {
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
	return runPolicyCheckUseCase(ctx, runtime, policyengine.CheckRequest{ModulePath: modulePath}, outputFmt, w)
}
