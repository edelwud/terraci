package policy

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func runPullPoliciesUseCase(ctx context.Context, runtime *policyRuntime) error {
	dirs, err := runtime.puller.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull policies: %w", err)
	}

	log.WithField("count", len(dirs)).Info("policy sources pulled")
	log.WithField("cache", runtime.puller.CacheDir()).Info("policies cached")
	return nil
}

func runPolicyCheckUseCase(ctx context.Context, runtime *policyRuntime, w io.Writer) error {
	policyDirs, err := runtime.puller.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull policies: %w", err)
	}

	checker := policyengine.NewChecker(runtime.config, policyDirs, runtime.workDir)
	summary, err := buildPolicySummary(ctx, checker, runtime.options.modulePath)
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}

	persistPolicyArtifacts(runtime.serviceDir, summary)
	return outputResult(w, runtime.options.outputFmt, summary, checker.ShouldBlock(summary))
}

func buildPolicySummary(ctx context.Context, checker *policyengine.Checker, modulePath string) (*policyengine.Summary, error) {
	if modulePath != "" {
		result, err := checker.CheckModule(ctx, modulePath)
		if err != nil {
			return nil, err
		}
		return policyengine.NewSummary([]policyengine.Result{*result}), nil
	}

	return checker.CheckAll(ctx)
}

func persistPolicyArtifacts(serviceDir string, summary *policyengine.Summary) {
	if serviceDir == "" {
		return
	}

	if saveErr := ci.SaveJSON(serviceDir, resultsFile, summary); saveErr != nil {
		log.WithError(saveErr).Warn("failed to save policy results")
	}
	if saveErr := ci.SaveReport(serviceDir, buildPolicyReport(summary)); saveErr != nil {
		log.WithError(saveErr).Warn("failed to save policy report")
	}
}

func (p *Plugin) runPull(ctx context.Context, appCtx *plugin.AppContext, outputDir string) error {
	runtime, err := p.runtime(ctx, appCtx, runtimeOptions{outputDir: outputDir})
	if err != nil {
		return err
	}

	return runPullPoliciesUseCase(ctx, runtime)
}

func (p *Plugin) runCheck(ctx context.Context, appCtx *plugin.AppContext, modulePath, outputFmt string) error {
	runtime, err := p.runtime(ctx, appCtx, runtimeOptions{
		modulePath: modulePath,
		outputFmt:  outputFmt,
	})
	if err != nil {
		return err
	}

	return runPolicyCheckUseCase(ctx, runtime, os.Stdout)
}
