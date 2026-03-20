package gitlab

import (
	"github.com/edelwud/terraci/internal/ci"
)

// Re-export shared types for backward compatibility
type (
	PlanResult           = ci.PlanResult
	PlanResultCollection = ci.PlanResultCollection
)

// ScanPlanResults wraps the shared implementation
var ScanPlanResults = ci.ScanPlanResults

// FormatPlanSummary wraps the shared implementation
var FormatPlanSummary = ci.FormatPlanSummary

// FormatPlanDetails wraps the shared implementation
var FormatPlanDetails = ci.FormatPlanDetails

// FilterPlanOutput wraps the shared implementation
var FilterPlanOutput = ci.FilterPlanOutput
