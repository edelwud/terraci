package cost

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/planresults"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

type planDiscovery struct {
	collection  *ci.PlanResultCollection
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

	if err := saveArtifacts(ctx, appCtx, result, plans.collection); err != nil {
		log.WithError(err).Warn("cost: failed to save artifacts")
	}

	return outputResult(w, appCtx.WorkDir(), outputFmt, result)
}

func discoverModulePlans(appCtx *plugin.AppContext, modulePath string) (*planDiscovery, error) {
	cfg := appCtx.Config()
	workDir := appCtx.WorkDir()

	log.WithField("dir", workDir).Info("cost: scanning for plan.json files")

	collection, err := planresults.Scan(workDir, cfg.Structure.Segments)
	if err != nil {
		return nil, fmt.Errorf("cost: scan for plan.json: %w", err)
	}

	paths := planModulePaths(workDir, collection)
	paths = filterModulePaths(workDir, paths, modulePath)
	if len(paths) == 0 {
		return nil, fmt.Errorf("cost: no plan.json files found in %s", workDir)
	}

	log.WithField("count", len(paths)).Info("cost: modules with plan.json found")

	regions := make(map[string]string, len(paths))
	plansByPath := planResultsByFullPath(workDir, collection)
	for _, fullPath := range paths {
		if plan, ok := plansByPath[fullPath]; ok {
			if region := plan.Get("region"); region != "" {
				regions[fullPath] = region
				continue
			}
		}
		if relDir, relErr := filepath.Rel(workDir, fullPath); relErr == nil {
			regions[fullPath] = model.DetectRegion(cfg.Structure.Segments, filepath.ToSlash(relDir))
		}
	}

	return &planDiscovery{
		collection:  collection,
		modulePaths: paths,
		regions:     regions,
	}, nil
}

func planModulePaths(workDir string, collection *ci.PlanResultCollection) []string {
	if collection == nil {
		return nil
	}
	paths := make([]string, 0, len(collection.Results))
	for i := range collection.Results {
		modulePath := filepath.FromSlash(collection.Results[i].ModulePath)
		paths = append(paths, filepath.Join(workDir, modulePath))
	}
	return paths
}

func planResultsByFullPath(workDir string, collection *ci.PlanResultCollection) map[string]*ci.PlanResult {
	results := make(map[string]*ci.PlanResult)
	if collection == nil {
		return results
	}
	for i := range collection.Results {
		fullPath := filepath.Join(workDir, filepath.FromSlash(collection.Results[i].ModulePath))
		results[fullPath] = &collection.Results[i]
	}
	return results
}

func filterModulePaths(workDir string, modulePaths []string, modulePath string) []string {
	if modulePath == "" {
		return modulePaths
	}

	target := filepath.Join(workDir, filepath.FromSlash(modulePath))
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
