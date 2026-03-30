package results

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

// EstimateAction is the result-assembly action model shared with the engine.
type EstimateAction string

const (
	ActionCreate  EstimateAction = "create"
	ActionDelete  EstimateAction = "delete"
	ActionUpdate  EstimateAction = "update"
	ActionReplace EstimateAction = "replace"
	ActionNoOp    EstimateAction = "no-op"
)

// ModuleIdentity identifies the module result being assembled.
type ModuleIdentity struct {
	ModuleID   string
	ModulePath string
	Region     string
	HasChanges bool
}

// ModuleAssembler builds a module-level result from resolved resources.
type ModuleAssembler struct {
	result      model.ModuleCost
	providerSet map[string]bool
}

// NewModuleAssembler creates a module result assembler for a single module.
func NewModuleAssembler(identity ModuleIdentity) *ModuleAssembler {
	return &ModuleAssembler{
		result: model.ModuleCost{
			ModuleID:   identity.ModuleID,
			ModulePath: identity.ModulePath,
			Region:     identity.Region,
			Resources:  make([]model.ResourceCost, 0),
			HasChanges: identity.HasChanges,
		},
		providerSet: make(map[string]bool),
	}
}

// AddResource adds a resolved resource and updates the module totals.
func (a *ModuleAssembler) AddResource(rc model.ResourceCost, action EstimateAction) {
	a.result.Resources = append(a.result.Resources, rc)
	if rc.Provider != "" {
		a.providerSet[rc.Provider] = true
	}
	AggregateCost(&a.result, rc, action)
}

// Build finalizes and returns the assembled module result.
func (a *ModuleAssembler) Build() *model.ModuleCost {
	a.result.DiffCost = a.result.AfterCost - a.result.BeforeCost
	a.result.Submodules = model.GroupByModule(a.result.Resources)
	a.result.Provider, a.result.Providers = summarizeProviders(a.providerSet)
	return &a.result
}

// AggregateCost applies one resource contribution to the module totals.
func AggregateCost(result *model.ModuleCost, rc model.ResourceCost, action EstimateAction) {
	if rc.IsUnsupported() {
		result.Unsupported++
		return
	}

	switch action {
	case ActionCreate:
		result.AfterCost += rc.MonthlyCost
	case ActionDelete:
		result.BeforeCost += rc.MonthlyCost
	case ActionUpdate, ActionReplace:
		result.BeforeCost += rc.BeforeMonthlyCost
		result.AfterCost += rc.MonthlyCost
	case ActionNoOp:
		result.BeforeCost += rc.MonthlyCost
		result.AfterCost += rc.MonthlyCost
	}
}

// EstimateAssembler builds the final estimate result from ordered module results.
type EstimateAssembler struct {
	modules          []model.ModuleCost
	providerMetadata map[string]model.ProviderMetadata
	generatedAt      time.Time
}

// NewEstimateAssembler creates a result assembler for a multi-module estimate.
func NewEstimateAssembler(providerMetadata map[string]model.ProviderMetadata, generatedAt time.Time) *EstimateAssembler {
	return &EstimateAssembler{
		modules:          make([]model.ModuleCost, 0),
		providerMetadata: providerMetadata,
		generatedAt:      generatedAt,
	}
}

// AddModule appends one module result in its final output order.
func (a *EstimateAssembler) AddModule(module model.ModuleCost) {
	a.modules = append(a.modules, module)
}

// Build finalizes and returns the estimate result.
func (a *EstimateAssembler) Build() *model.EstimateResult {
	result := &model.EstimateResult{
		Modules:          append([]model.ModuleCost(nil), a.modules...),
		Currency:         "USD",
		GeneratedAt:      a.generatedAt,
		ProviderMetadata: a.providerMetadata,
	}

	providerSet := make(map[string]bool)
	for i := range a.modules {
		module := &a.modules[i]
		result.TotalBefore += module.BeforeCost
		result.TotalAfter += module.AfterCost
		for _, providerID := range module.Providers {
			providerSet[providerID] = true
		}
		if module.Error != "" {
			result.Errors = append(result.Errors, model.ModuleError{
				ModuleID: module.ModuleID,
				Error:    module.Error,
			})
		}
	}
	result.TotalDiff = result.TotalAfter - result.TotalBefore
	result.Providers = sortedProviderIDs(providerSet)

	return result
}

// NewErroredModule builds a stable module result for a scan or load failure.
func NewErroredModule(modulePath, region string, err error) model.ModuleCost {
	return model.ModuleCost{
		ModuleID:   strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath: modulePath,
		Region:     region,
		Error:      err.Error(),
	}
}

func summarizeProviders(providerSet map[string]bool) (primary string, providers []string) {
	if len(providerSet) == 0 {
		return "", nil
	}

	providers = sortedProviderIDs(providerSet)
	if len(providers) == 1 {
		return providers[0], providers
	}
	return "", providers
}

func sortedProviderIDs(providerSet map[string]bool) []string {
	providers := make([]string, 0, len(providerSet))
	for providerID := range providerSet {
		providers = append(providers, providerID)
	}
	slices.Sort(providers)
	return providers
}
