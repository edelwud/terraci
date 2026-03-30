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

func runPullPolicies(ctx context.Context, appCtx *plugin.AppContext, cfg *policyengine.Config, outputDir string) error {
	workDir := appCtx.WorkDir()
	serviceDir := appCtx.ServiceDir()
	if outputDir != "" {
		cfg.CacheDir = outputDir
	}

	puller, err := policyengine.NewPuller(cfg, workDir, serviceDir)
	if err != nil {
		return fmt.Errorf("failed to create puller: %w", err)
	}

	dirs, err := puller.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull policies: %w", err)
	}

	log.WithField("count", len(dirs)).Info("policy sources pulled")
	log.WithField("cache", puller.CacheDir()).Info("policies cached")
	return nil
}

func runPolicyCheck(ctx context.Context, appCtx *plugin.AppContext, cfg *policyengine.Config, modulePath, outputFmt string, w io.Writer) error {
	workDir := appCtx.WorkDir()
	serviceDir := appCtx.ServiceDir()

	puller, err := policyengine.NewPuller(cfg, workDir, serviceDir)
	if err != nil {
		return fmt.Errorf("failed to create puller: %w", err)
	}

	policyDirs, err := puller.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull policies: %w", err)
	}

	checker := policyengine.NewChecker(cfg, policyDirs, workDir)
	summary, err := buildPolicySummary(ctx, checker, modulePath)
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}

	persistPolicyArtifacts(serviceDir, summary)
	return outputResult(w, outputFmt, summary, checker.ShouldBlock(summary))
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

func (p *Plugin) runCheck(ctx context.Context, appCtx *plugin.AppContext, modulePath, outputFmt string) error {
	return runPolicyCheck(ctx, appCtx, p.Config(), modulePath, outputFmt, os.Stdout)
}
