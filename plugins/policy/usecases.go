package policy

import (
	"context"
	"fmt"
	"io"
	"os"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/policy/internal/domain"
	policyusecase "github.com/edelwud/terraci/plugins/policy/internal/usecase"
)

func runPullPoliciesUseCase(ctx context.Context, runtime *policyRuntime) error {
	result, err := policyusecase.Pull(ctx, runtime.sources, policyusecase.PullRequest{})
	if err != nil {
		return fmt.Errorf("failed to materialize policies: %w", err)
	}

	log.WithField("count", len(result.PolicyDirs)).Info("policy sources materialized")
	log.WithField("cache", result.CacheDir).Info("policies cached")
	return nil
}

func runPolicyCheckUseCase(ctx context.Context, runtime *policyRuntime, w io.Writer) error {
	summary, err := policyusecase.Check(ctx, policyusecase.CheckRuntime{
		Config:       runtime.config,
		Sources:      runtime.sources,
		WorkDir:      runtime.workDir,
		PlanSegments: runtime.planSegments,
	}, policyusecase.CheckRequest{ModulePath: runtime.options.modulePath})
	if err != nil {
		return fmt.Errorf("run policy check: %w", err)
	}

	persistPolicyArtifacts(runtime.serviceDir, summary)
	return outputResult(w, runtime.options.outputFmt, summary, summary.HasFailures())
}

func persistPolicyArtifacts(serviceDir string, summary *domain.Summary) {
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

func (p *Plugin) runPull(ctx context.Context, appCtx *plugin.AppContext, outputDir string) error {
	runtime, err := p.runtime(ctx, appCtx, &runtimeOptions{outputDir: outputDir})
	if err != nil {
		return err
	}

	return runPullPoliciesUseCase(ctx, runtime)
}

func (p *Plugin) runCheck(ctx context.Context, appCtx *plugin.AppContext, modulePath, outputFmt string) error {
	runtime, err := p.runtime(ctx, appCtx, &runtimeOptions{
		modulePath: modulePath,
		outputFmt:  outputFmt,
	})
	if err != nil {
		return err
	}

	return runPolicyCheckUseCase(ctx, runtime, os.Stdout)
}
