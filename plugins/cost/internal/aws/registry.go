// Package aws provides AWS-specific resource cost estimation handlers.
package aws

import "github.com/edelwud/terraci/plugins/cost/internal/provider"

// Type aliases for backward compatibility — all handlers import from aws/,
// but the actual definitions live in provider/ for multi-provider support.
type (
	CostCategory    = provider.CostCategory
	ResourceHandler = provider.ResourceHandler
	CompoundHandler = provider.CompoundHandler
	SubResource     = provider.SubResource
	Registry        = provider.Registry
)

// Cost category constants re-exported for aws/ handlers.
const (
	CostCategoryStandard   = provider.CostCategoryStandard
	CostCategoryFixed      = provider.CostCategoryFixed
	CostCategoryUsageBased = provider.CostCategoryUsageBased
)

// HoursPerMonth re-exported for aws/ handlers.
const HoursPerMonth = provider.HoursPerMonth

// Attribute helpers re-exported for aws/ handlers.
var (
	GetStringAttr = provider.GetStringAttr
	GetFloatAttr  = provider.GetFloatAttr
	GetIntAttr    = provider.GetIntAttr
	GetBoolAttr   = provider.GetBoolAttr
)

// Cost calculation helpers re-exported for aws/ handlers.
var (
	HourlyCost       = provider.HourlyCost
	ScaledHourlyCost = provider.ScaledHourlyCost
	FixedMonthlyCost = provider.FixedMonthlyCost
)

// LogUnsupported re-exported.
var LogUnsupported = provider.LogUnsupported

// NewRegistryEmpty creates a new empty registry (for testing).
func NewRegistryEmpty() *Registry {
	return provider.NewRegistry()
}

// NewRegistry creates a new registry with all AWS handlers registered.
// RegisterAll must be set before calling NewRegistry.
func NewRegistry() *Registry {
	InitRegionMapping()
	r := provider.NewRegistry()
	if RegisterAll != nil {
		RegisterAll(r)
	}
	return r
}

// RegisterAll registers all built-in AWS resource handlers from subpackages.
// Called by the cost/internal/registry.go init() to avoid import cycles.
var RegisterAll func(r *Registry)
