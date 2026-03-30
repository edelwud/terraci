package cost

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

type estimateInputs struct {
	modulePaths []string
	regions     map[string]string
}

func runEstimationUseCase(ctx context.Context, appCtx *plugin.AppContext, runtime *costRuntime, modulePath, outputFmt string, w io.Writer) error {
	inputs, err := discoverEstimateInputs(appCtx, modulePath)
	if err != nil {
		return err
	}

	result, err := runtime.estimator.EstimateModules(ctx, inputs.modulePaths, inputs.regions)
	if err != nil {
		return fmt.Errorf("estimate costs: %w", err)
	}

	persistEstimateArtifacts(appCtx.ServiceDir(), result)

	return outputResult(w, appCtx.WorkDir(), outputFmt, result)
}

func discoverEstimateInputs(appCtx *plugin.AppContext, modulePath string) (*estimateInputs, error) {
	cfg := appCtx.Config()
	workDir := appCtx.WorkDir()

	log.WithField("dir", workDir).Info("scanning for plan.json files")

	modulePaths, err := discovery.FindModulesWithPlan(workDir)
	if err != nil {
		return nil, fmt.Errorf("scan for plan.json: %w", err)
	}

	modulePaths = filterModulePaths(workDir, modulePaths, modulePath)
	if len(modulePaths) == 0 {
		return nil, errors.New("no plan.json files found")
	}

	log.WithField("count", len(modulePaths)).Info("modules with plan.json found")

	regions := make(map[string]string, len(modulePaths))
	for _, fullPath := range modulePaths {
		relDir, relErr := filepath.Rel(workDir, fullPath)
		if relErr == nil {
			regions[fullPath] = model.DetectRegion(cfg.Structure.Segments, relDir)
		}
	}

	return &estimateInputs{
		modulePaths: modulePaths,
		regions:     regions,
	}, nil
}

func filterModulePaths(workDir string, modulePaths []string, modulePath string) []string {
	if modulePath == "" {
		return modulePaths
	}

	target := filepath.Join(workDir, modulePath)
	filtered := make([]string, 0, 1)
	for _, path := range modulePaths {
		if path == target || strings.HasSuffix(path, modulePath) {
			filtered = append(filtered, path)
		}
	}

	return filtered
}

func persistEstimateArtifacts(serviceDir string, result *model.EstimateResult) {
	if serviceDir == "" {
		return
	}

	if saveErr := ci.SaveJSON(serviceDir, resultsFile, result); saveErr != nil {
		log.WithError(saveErr).Warn("failed to save cost results")
	}

	report := buildCostReport(result)
	if saveErr := ci.SaveReport(serviceDir, report); saveErr != nil {
		log.WithError(saveErr).Warn("failed to save cost report")
	}
}

func (p *Plugin) runEstimation(ctx context.Context, appCtx *plugin.AppContext, modulePath, outputFmt string) error {
	return p.runEstimationWithWriter(ctx, appCtx, modulePath, outputFmt, os.Stdout)
}

func (p *Plugin) runEstimationWithWriter(ctx context.Context, appCtx *plugin.AppContext, modulePath, outputFmt string, w io.Writer) error {
	runtime, err := newRuntime(p.Config())
	if err != nil {
		return err
	}

	return runEstimationUseCase(ctx, appCtx, runtime, modulePath, outputFmt, w)
}
