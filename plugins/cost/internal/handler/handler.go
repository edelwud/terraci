// Package handler provides backward-compatible type aliases for the canonical
// types that now live in resourcedef. Import this package when you need
// ResourceType, CostCategory, or SubResource without changing existing imports.
package handler

import "github.com/edelwud/terraci/plugins/cost/internal/resourcedef"

// ResourceType is an alias for resourcedef.ResourceType.
type ResourceType = resourcedef.ResourceType

// CostCategory is an alias for resourcedef.CostCategory.
type CostCategory = resourcedef.CostCategory

// SubResource is an alias for resourcedef.SubResource.
type SubResource = resourcedef.SubResource

const (
	CostCategoryStandard   = resourcedef.CostCategoryStandard
	CostCategoryFixed      = resourcedef.CostCategoryFixed
	CostCategoryUsageBased = resourcedef.CostCategoryUsageBased
)
