package results

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
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
	providerSet map[string]struct{}
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
		providerSet: make(map[string]struct{}),
	}
}

// AddResource adds a resolved resource and updates the module totals.
func (a *ModuleAssembler) AddResource(rc model.ResourceCost, action model.EstimateAction) {
	a.result.Resources = append(a.result.Resources, rc)
	if rc.Provider != "" {
		a.providerSet[rc.Provider] = struct{}{}
	}
	AggregateCost(&a.result, rc, action)
}

// Build finalizes and returns the assembled module result.
func (a *ModuleAssembler) Build() *model.ModuleCost {
	a.result.DiffCost = a.result.AfterCost - a.result.BeforeCost
	a.result.Provider, a.result.Providers = summarizeProviders(a.providerSet)
	return &a.result
}

// AggregateCost applies one resource contribution to the module totals.
func AggregateCost(result *model.ModuleCost, rc model.ResourceCost, action model.EstimateAction) {
	switch rc.Status {
	case model.ResourceEstimateStatusUnsupported:
		result.Unsupported++
		return
	case model.ResourceEstimateStatusUsageEstimated:
		result.UsageEstimated++
	case model.ResourceEstimateStatusUsageUnknown:
		result.UsageUnknown++
		return
	case model.ResourceEstimateStatusFailed:
		return
	case model.ResourceEstimateStatusExact:
		// handled below
	default:
		return
	}

	if !rc.ContributesAfterCost() {
		return
	}

	if rc.Status == model.ResourceEstimateStatusUsageEstimated {
		switch action {
		case model.ActionCreate, model.ActionUpdate, model.ActionReplace, model.ActionNoOp:
			result.AfterCost += rc.MonthlyCost
		case model.ActionDelete:
			result.BeforeCost += rc.MonthlyCost
		}
		return
	}

	switch action {
	case model.ActionCreate:
		result.AfterCost += rc.MonthlyCost
	case model.ActionDelete:
		result.BeforeCost += rc.MonthlyCost
	case model.ActionUpdate, model.ActionReplace:
		result.BeforeCost += rc.BeforeMonthlyCost
		result.AfterCost += rc.MonthlyCost
	case model.ActionNoOp:
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
		Modules:          slices.Clone(a.modules),
		Currency:         "USD",
		GeneratedAt:      a.generatedAt,
		ProviderMetadata: a.providerMetadata,
	}

	providerSet := make(map[string]struct{})
	for i := range a.modules {
		module := &a.modules[i]
		result.TotalBefore += module.BeforeCost
		result.TotalAfter += module.AfterCost
		result.Unsupported += module.Unsupported
		result.UsageEstimated += module.UsageEstimated
		result.UsageUnknown += module.UsageUnknown
		for _, providerID := range module.Providers {
			providerSet[providerID] = struct{}{}
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
		Resources:  []model.ResourceCost{},
	}
}

func summarizeProviders(providerSet map[string]struct{}) (primary string, providers []string) {
	if len(providerSet) == 0 {
		return "", nil
	}

	providers = sortedProviderIDs(providerSet)
	if len(providers) == 1 {
		return providers[0], providers
	}
	return "", providers
}

func sortedProviderIDs(providerSet map[string]struct{}) []string {
	providers := make([]string, 0, len(providerSet))
	for providerID := range providerSet {
		providers = append(providers, providerID)
	}
	slices.Sort(providers)
	return providers
}
