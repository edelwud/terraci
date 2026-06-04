package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/planresults"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
	"github.com/edelwud/terraci/plugins/policy/internal/engine"
	policyinput "github.com/edelwud/terraci/plugins/policy/internal/input"
)

type SourceMaterializer interface {
	Materialize(ctx context.Context, cacheDirOverride string) ([]string, error)
	CacheDir(cacheDirOverride string) string
}

type PlanScanner interface {
	Scan(rootDir string, segments []string) (*ci.PlanResultCollection, error)
}

type Evaluator interface {
	Evaluate(ctx context.Context, input policyinput.Envelope, namespaces policyengine.Namespaces) (*policyengine.Evaluation, error)
}

type EvaluatorFactory interface {
	NewEvaluator(policyDirs []string) Evaluator
}

type CheckRuntime struct {
	Config           *policyengine.Config
	Sources          SourceMaterializer
	PlanScanner      PlanScanner
	EvaluatorFactory EvaluatorFactory
	WorkDir          string
	PlanSegments     []string
}

type CheckResult struct {
	Summary     *policyengine.Summary
	PlanResults *ci.PlanResultCollection
}

func Check(ctx context.Context, runtime CheckRuntime, req policyengine.CheckRequest) (*CheckResult, error) {
	if runtime.Config == nil {
		return nil, errors.New("policy config is nil")
	}
	if runtime.Sources == nil {
		return nil, errors.New("policy source materializer is nil")
	}

	policyDirs, err := runtime.Sources.Materialize(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("materialize policy sources: %w", err)
	}

	collection, plans, err := discoverPlans(runtime.PlanScanner, runtime.WorkDir, runtime.PlanSegments, req.ModulePath)
	if err != nil {
		return nil, err
	}

	evaluatorFactory := runtime.EvaluatorFactory
	if evaluatorFactory == nil {
		evaluatorFactory = engineFactory{}
	}
	evaluator := evaluatorFactory.NewEvaluator(policyDirs)

	results := make([]policyengine.Result, 0, len(plans))
	for i := range plans {
		results = append(results, checkPlan(ctx, runtime, evaluator, plans[i]))
	}

	return &CheckResult{
		Summary:     policyengine.NewSummary(results),
		PlanResults: collection,
	}, nil
}

func discoverPlans(scanner PlanScanner, workDir string, segments []string, modulePath string) (*ci.PlanResultCollection, []ci.PlanResult, error) {
	if scanner == nil {
		scanner = defaultPlanScanner{}
	}

	collection, err := scanner.Scan(workDir, segments)
	if err != nil {
		return nil, nil, fmt.Errorf("scan plan results: %w", err)
	}
	if collection == nil {
		return nil, nil, errors.New("scan plan results: nil collection")
	}

	plans := collection.Results()
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].ModulePath() < plans[j].ModulePath()
	})

	filtered := filterPlans(plans, modulePath)
	if len(filtered) == 0 {
		if modulePath != "" {
			return nil, nil, fmt.Errorf("no plan.json found for module %q", modulePath)
		}
		return nil, nil, fmt.Errorf("no plan.json files found in %s", workDir)
	}
	return collection, filtered, nil
}

func filterPlans(plans []ci.PlanResult, modulePath string) []ci.PlanResult {
	if modulePath == "" {
		return plans
	}

	modulePath = filepath.ToSlash(modulePath)
	filtered := make([]ci.PlanResult, 0, len(plans))
	for i := range plans {
		plan := plans[i]
		planPath := filepath.ToSlash(plan.ModulePath())
		if planPath == modulePath || strings.HasSuffix(planPath, "/"+modulePath) {
			filtered = append(filtered, plan)
		}
	}
	return filtered
}

func checkPlan(ctx context.Context, runtime CheckRuntime, evaluator Evaluator, plan ci.PlanResult) policyengine.Result {
	modulePath := filepath.ToSlash(plan.ModulePath())
	effective, err := runtime.Config.EffectiveConfig(modulePath)
	if err != nil {
		return policyengine.NewErrorResult(modulePath, err)
	}
	if effective == nil || !effective.Enabled {
		return policyengine.NewSkippedResult(modulePath)
	}

	namespaces := effective.NamespacesOrDefault()
	planJSONPath := filepath.Join(runtime.WorkDir, filepath.FromSlash(modulePath), pipeline.PlanJSONFilename)
	envelope, err := policyinput.Build(policyinput.Request{
		PlanJSONPath:    planJSONPath,
		PlanDisplayPath: filepath.ToSlash(filepath.Join(modulePath, pipeline.PlanJSONFilename)),
		ModulePath:      modulePath,
		Components:      plan.Components(),
		Namespaces:      namespaces,
	})
	if err != nil {
		return policyengine.NewErrorResult(modulePath, err)
	}

	if evaluator == nil {
		return policyengine.NewErrorResult(modulePath, errors.New("policy evaluator is nil"))
	}
	evaluation, err := evaluator.Evaluate(ctx, envelope, namespaces)
	if err != nil {
		return policyengine.NewErrorResult(modulePath, err)
	}

	return policyengine.ApplyEvaluation(modulePath, evaluation, effective.Decisions)
}

type defaultPlanScanner struct{}

func (defaultPlanScanner) Scan(rootDir string, segments []string) (*ci.PlanResultCollection, error) {
	return planresults.Scan(rootDir, segments)
}

type engineFactory struct{}

func (engineFactory) NewEvaluator(policyDirs []string) Evaluator {
	return engine.New(policyDirs)
}
