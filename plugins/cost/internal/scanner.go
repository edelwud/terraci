package costengine

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
)

// ModuleScanner converts Terraform plan files into internal module plans.
type ModuleScanner struct{}

// NewModuleScanner creates a module scanner.
func NewModuleScanner() *ModuleScanner {
	return &ModuleScanner{}
}

// PlannedResource is the scanner's internal resource IR decoupled from raw Terraform plan types.
type PlannedResource struct {
	ResourceType handler.ResourceType
	Address      string
	Name         string
	ModuleAddr   string
	Action       string
	BeforeAttrs  map[string]any
	AfterAttrs   map[string]any
}

// ResolveRequest returns the primary resolution request for this planned resource.
func (r PlannedResource) ResolveRequest(region string) ResolveRequest {
	return ResolveRequest{
		ResourceType: r.ResourceType,
		Address:      r.Address,
		Name:         r.Name,
		ModuleAddr:   r.ModuleAddr,
		Region:       region,
		Attrs:        r.ActiveAttrs(),
	}
}

// ActiveAttrs returns the attrs that represent the resource's current target state.
func (r PlannedResource) ActiveAttrs() map[string]any {
	if r.AfterAttrs != nil {
		return r.AfterAttrs
	}
	return r.BeforeAttrs
}

// RequiresBeforeCost reports whether the before-state should be priced separately.
func (r PlannedResource) RequiresBeforeCost() bool {
	return (r.Action == plan.ActionUpdate || r.Action == plan.ActionReplace) && r.BeforeAttrs != nil
}

// ModulePlan is the scanner's internal module IR produced from Terraform plan JSON.
type ModulePlan struct {
	ModuleID   string
	ModulePath string
	Region     string
	HasChanges bool
	Resources  []PlannedResource
}

// Scan reads a module plan from disk and converts it into the estimator IR.
func (s *ModuleScanner) Scan(modulePath, region string) (*ModulePlan, error) {
	planJSONPath := filepath.Join(modulePath, "plan.json")

	parsedPlan, err := plan.ParseJSON(planJSONPath)
	if err != nil {
		return nil, fmt.Errorf("parse plan.json: %w", err)
	}

	modulePlan := &ModulePlan{
		ModuleID:   strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath: modulePath,
		Region:     region,
		HasChanges: parsedPlan.HasChanges(),
		Resources:  make([]PlannedResource, 0, len(parsedPlan.Resources)),
	}

	for _, rc := range parsedPlan.Resources {
		modulePlan.Resources = append(modulePlan.Resources, PlannedResource{
			ResourceType: handler.ResourceType(rc.Type),
			Address:      rc.Address,
			Name:         rc.Name,
			ModuleAddr:   rc.ModuleAddr,
			Action:       rc.Action,
			BeforeAttrs:  rc.BeforeValues,
			AfterAttrs:   rc.AfterValues,
		})
	}

	return modulePlan, nil
}

// ScanMany scans multiple modules strictly, returning an error on the first failure.
func (s *ModuleScanner) ScanMany(modulePaths []string, regions map[string]string) ([]*ModulePlan, error) {
	plans := make([]*ModulePlan, 0, len(modulePaths))
	for _, modulePath := range modulePaths {
		region := regions[modulePath]
		if region == "" {
			region = DefaultRegion
		}

		modulePlan, err := s.Scan(modulePath, region)
		if err != nil {
			return nil, err
		}
		plans = append(plans, modulePlan)
	}
	return plans, nil
}

// ScannedModulePlan captures either a scanned plan or a per-module scan error.
type ScannedModulePlan struct {
	Index      int
	ModulePath string
	Region     string
	Plan       *ModulePlan
	Err        error
}

// ScanManyBestEffort scans multiple modules and preserves per-module failures.
func (s *ModuleScanner) ScanManyBestEffort(modulePaths []string, regions map[string]string) []ScannedModulePlan {
	plans := make([]ScannedModulePlan, 0, len(modulePaths))
	for i, modulePath := range modulePaths {
		region := regions[modulePath]
		if region == "" {
			region = DefaultRegion
		}

		modulePlan, err := s.Scan(modulePath, region)
		plans = append(plans, ScannedModulePlan{
			Index:      i,
			ModulePath: modulePath,
			Region:     region,
			Plan:       modulePlan,
			Err:        err,
		})
	}
	return plans
}
