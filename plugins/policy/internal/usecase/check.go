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
	policyconfig "github.com/edelwud/terraci/plugins/policy/internal/config"
	"github.com/edelwud/terraci/plugins/policy/internal/domain"
	"github.com/edelwud/terraci/plugins/policy/internal/engine"
	policyinput "github.com/edelwud/terraci/plugins/policy/internal/input"
	"github.com/edelwud/terraci/plugins/policy/internal/source"
)

type CheckRequest struct {
	ModulePath string
}

type CheckRuntime struct {
	Config       *policyconfig.Config
	Sources      *source.Materializer
	WorkDir      string
	PlanSegments []string
}

func Check(ctx context.Context, runtime CheckRuntime, req CheckRequest) (*domain.Summary, error) {
	if runtime.Config == nil {
		return nil, errors.New("policy config is nil")
	}
	if runtime.Sources == nil {
		return nil, errors.New("policy source materializer is nil")
	}

	policyDirs, err := runtime.Sources.Materialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("materialize policy sources: %w", err)
	}

	plans, err := discoverPlans(runtime.WorkDir, runtime.PlanSegments, req.ModulePath)
	if err != nil {
		return nil, err
	}

	results := make([]domain.Result, 0, len(plans))
	for i := range plans {
		results = append(results, checkPlan(ctx, runtime, policyDirs, plans[i]))
	}

	return domain.NewSummary(results), nil
}

func discoverPlans(workDir string, segments []string, modulePath string) ([]ci.PlanResult, error) {
	collection, err := planresults.Scan(workDir, segments)
	if err != nil {
		return nil, fmt.Errorf("scan plan results: %w", err)
	}

	plans := collection.Results
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].ModulePath < plans[j].ModulePath
	})

	filtered := filterPlans(plans, modulePath)
	if len(filtered) == 0 {
		if modulePath != "" {
			return nil, fmt.Errorf("no plan.json found for module %q", modulePath)
		}
		return nil, fmt.Errorf("no plan.json files found in %s", workDir)
	}
	return filtered, nil
}

func filterPlans(plans []ci.PlanResult, modulePath string) []ci.PlanResult {
	if modulePath == "" {
		return plans
	}

	modulePath = filepath.ToSlash(modulePath)
	filtered := make([]ci.PlanResult, 0, len(plans))
	for i := range plans {
		plan := &plans[i]
		planPath := filepath.ToSlash(plan.ModulePath)
		if planPath == modulePath || strings.HasSuffix(planPath, "/"+modulePath) {
			filtered = append(filtered, *plan)
		}
	}
	return filtered
}

func checkPlan(ctx context.Context, runtime CheckRuntime, policyDirs []string, plan ci.PlanResult) domain.Result {
	modulePath := filepath.ToSlash(plan.ModulePath)
	effective, err := runtime.Config.EffectiveConfig(modulePath)
	if err != nil {
		return domain.NewErrorResult(modulePath, err)
	}
	if effective == nil || !effective.Enabled {
		return domain.NewSkippedResult(modulePath)
	}

	namespaces := effective.NamespacesOrDefault()
	planJSONPath := filepath.Join(runtime.WorkDir, filepath.FromSlash(modulePath), pipeline.PlanJSONFilename)
	envelope, err := policyinput.Build(policyinput.Request{
		PlanJSONPath:    planJSONPath,
		PlanDisplayPath: filepath.ToSlash(filepath.Join(modulePath, pipeline.PlanJSONFilename)),
		ModulePath:      modulePath,
		Components:      plan.Components,
		Namespaces:      namespaces,
	})
	if err != nil {
		return domain.NewErrorResult(modulePath, err)
	}

	evaluation, err := engine.New(policyDirs, namespaces).Evaluate(ctx, envelope)
	if err != nil {
		return domain.NewErrorResult(modulePath, err)
	}

	return domain.ApplyEvaluation(modulePath, evaluation, effective.ActionPolicy())
}
