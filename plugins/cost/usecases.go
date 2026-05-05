package cost

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/planresults"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

type planDiscovery struct {
	modulePaths []string
	regions     map[string]string
}

func runEstimationUseCase(ctx context.Context, appCtx *plugin.AppContext, runtime *costRuntime, modulePath, outputFmt string, w io.Writer) error {
	plans, err := discoverModulePlans(appCtx, modulePath)
	if err != nil {
		return err
	}

	result, err := runtime.estimator.EstimateModules(ctx, plans.modulePaths, plans.regions)
	if err != nil {
		return fmt.Errorf("cost: estimate costs: %w", err)
	}

	if err := saveArtifacts(appCtx.ServiceDir(), result); err != nil {
		log.WithError(err).Warn("cost: failed to save artifacts")
	}

	return outputResult(w, appCtx.WorkDir(), outputFmt, result)
}

func discoverModulePlans(appCtx *plugin.AppContext, modulePath string) (*planDiscovery, error) {
	cfg := appCtx.Config()
	workDir := appCtx.WorkDir()

	log.WithField("dir", workDir).Info("cost: scanning for plan.json files")

	paths, err := planresults.FindModulesWithPlan(workDir)
	if err != nil {
		return nil, fmt.Errorf("cost: scan for plan.json: %w", err)
	}

	paths = filterModulePaths(workDir, paths, modulePath)
	if len(paths) == 0 {
		return nil, fmt.Errorf("cost: no plan.json files found in %s", workDir)
	}

	log.WithField("count", len(paths)).Info("cost: modules with plan.json found")

	regions := make(map[string]string, len(paths))
	for _, fullPath := range paths {
		relDir, relErr := filepath.Rel(workDir, fullPath)
		if relErr == nil {
			regions[fullPath] = model.DetectRegion(cfg.Structure.Segments, relDir)
		}
	}

	return &planDiscovery{
		modulePaths: paths,
		regions:     regions,
	}, nil
}

func filterModulePaths(workDir string, modulePaths []string, modulePath string) []string {
	if modulePath == "" {
		return modulePaths
	}

	target := filepath.Join(workDir, modulePath)
	var filtered []string
	for _, path := range modulePaths {
		if path == target || matchesModulePath(path, modulePath) {
			filtered = append(filtered, path)
		}
	}

	return filtered
}

// matchesModulePath reports whether path ends with the given modulePath segment,
// anchored to a path separator to prevent partial-segment false positives.
// For example, "other/foo/bar" does NOT match "foo/bar" — only exact suffix
// at a directory boundary counts.
func matchesModulePath(path, modulePath string) bool {
	p := filepath.ToSlash(path)
	m := filepath.ToSlash(modulePath)
	return strings.HasSuffix(p, "/"+m)
}

func (p *Plugin) runEstimation(ctx context.Context, appCtx *plugin.AppContext, modulePath, outputFmt string) error {
	return p.runEstimationWithWriter(ctx, appCtx, modulePath, outputFmt, os.Stdout)
}

func (p *Plugin) runEstimationWithWriter(ctx context.Context, appCtx *plugin.AppContext, modulePath, outputFmt string, w io.Writer) error {
	runtime, err := p.runtime(ctx, appCtx)
	if err != nil {
		return err
	}

	return runEstimationUseCase(ctx, appCtx, runtime, modulePath, outputFmt, w)
}
