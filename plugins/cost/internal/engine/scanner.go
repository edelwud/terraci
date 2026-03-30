package engine

import (
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/results"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

// DefaultRegion is used when no region is specified.
const DefaultRegion = model.DefaultRegion

// EstimateAction is the provider-neutral action model used by the cost engine.
type EstimateAction = results.EstimateAction

const (
	ActionCreate  = results.ActionCreate
	ActionDelete  = results.ActionDelete
	ActionUpdate  = results.ActionUpdate
	ActionReplace = results.ActionReplace
	ActionNoOp    = results.ActionNoOp
)

// ModulePlanAdapter converts external plan sources into the cost engine input model.
type ModulePlanAdapter interface {
	LoadModule(modulePath, region string) (*ModulePlan, error)
}

// ModuleScanner loads module plans through a source-specific adapter.
type ModuleScanner struct {
	adapter ModulePlanAdapter
}

// NewModuleScanner creates a module scanner for the provided plan adapter.
func NewModuleScanner(adapter ModulePlanAdapter) *ModuleScanner {
	return &ModuleScanner{adapter: adapter}
}

// PlannedResource is the scanner's internal resource IR decoupled from raw Terraform plan types.
type PlannedResource struct {
	ResourceType handler.ResourceType
	Address      string
	Name         string
	ModuleAddr   string
	Action       EstimateAction
	BeforeAttrs  map[string]any
	AfterAttrs   map[string]any
}

// ResolveRequest returns the primary resolution request for this planned resource.
func (r PlannedResource) ResolveRequest(region string) costruntime.ResolveRequest {
	return costruntime.ResolveRequest{
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
	return (r.Action == ActionUpdate || r.Action == ActionReplace) && r.BeforeAttrs != nil
}

// ModulePlan is the provider-neutral input model consumed by the cost engine.
type ModulePlan struct {
	ModuleID   string
	ModulePath string
	Region     string
	HasChanges bool
	Resources  []PlannedResource
}

// Scan loads a module plan through the configured adapter.
func (s *ModuleScanner) Scan(modulePath, region string) (*ModulePlan, error) {
	return s.adapter.LoadModule(modulePath, region)
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
