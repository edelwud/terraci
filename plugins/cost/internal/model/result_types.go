package model

import (
	"math"
	"time"
)

// costZeroThreshold is the epsilon used to treat accumulated float64 costs as zero.
// Costs below this threshold are considered effectively zero for display purposes.
const costZeroThreshold = 0.0001

// CostIsZero reports whether a cost value should be treated as zero for display.
// Uses an epsilon comparison to avoid false non-zero results from floating-point accumulation.
func CostIsZero(v float64) bool { return math.Abs(v) < costZeroThreshold }

// ResourceCost represents the estimated cost of a single resource.
type ResourceCost struct {
	Provider          string                 `json:"provider,omitempty"`
	Address           string                 `json:"address"`
	ModuleAddr        string                 `json:"module_addr,omitempty"`
	Type              string                 `json:"type"`
	Name              string                 `json:"name"`
	Region            string                 `json:"region"`
	MonthlyCost       float64                `json:"monthly_cost"`
	HourlyCost        float64                `json:"hourly_cost"`
	BeforeMonthlyCost float64                `json:"before_monthly_cost,omitempty"`
	BeforeHourlyCost  float64                `json:"before_hourly_cost,omitempty"`
	PriceSource       string                 `json:"price_source"`
	Status            ResourceEstimateStatus `json:"status"`
	FailureKind       FailureKind            `json:"failure_kind,omitempty"`
	StatusDetail      string                 `json:"status_detail,omitempty"`
	Details           map[string]string      `json:"details,omitempty"`
}

// IsUnsupported reports whether the resource is unsupported by the estimator.
func (r ResourceCost) IsUnsupported() bool {
	return r.Status == ResourceEstimateStatusUnsupported
}

// IsUsageBased reports whether the resource uses a usage-derived estimate path.
func (r ResourceCost) IsUsageBased() bool {
	return r.Status == ResourceEstimateStatusUsageEstimated || r.Status == ResourceEstimateStatusUsageUnknown
}

// IsFailed reports whether the resource estimation failed after resource-definition resolution.
func (r ResourceCost) IsFailed() bool {
	return r.Status == ResourceEstimateStatusFailed
}

// ContributesAfterCost reports whether this resource should contribute to after-cost totals.
func (r ResourceCost) ContributesAfterCost() bool {
	return r.Status == ResourceEstimateStatusExact || r.Status == ResourceEstimateStatusUsageEstimated
}

// ResourceEstimateStatus classifies the outcome of estimating a single resource.
type ResourceEstimateStatus string

const (
	ResourceEstimateStatusExact          ResourceEstimateStatus = "exact"
	ResourceEstimateStatusUsageEstimated ResourceEstimateStatus = "usage_estimated"
	ResourceEstimateStatusUsageUnknown   ResourceEstimateStatus = "usage_unknown"
	ResourceEstimateStatusUnsupported    ResourceEstimateStatus = "unsupported"
	ResourceEstimateStatusFailed         ResourceEstimateStatus = "failed"
)

// FailureKind classifies a real estimation failure or unsupported outcome.
type FailureKind string

const (
	FailureKindNone         FailureKind = ""
	FailureKindNoProvider   FailureKind = "no_provider"
	FailureKindNoHandler    FailureKind = "no_handler"
	FailureKindLookupFailed FailureKind = "lookup_failed"
	FailureKindAPIFailure   FailureKind = "api_failure"
	FailureKindNoPrice      FailureKind = "no_price"
	FailureKindInternal     FailureKind = "internal"
)

// UsageCostEstimate captures the plan-time estimate available for a usage-priced resource.
type UsageCostEstimate struct {
	HourlyCost  float64
	MonthlyCost float64
	Status      ResourceEstimateStatus
	Detail      string
}

// ModuleCost represents the total cost estimate for a terraform module.
type ModuleCost struct {
	ModuleID       string         `json:"module_id"`
	ModulePath     string         `json:"module_path"`
	Region         string         `json:"region"`
	Provider       string         `json:"provider,omitempty"`
	Providers      []string       `json:"providers,omitempty"`
	BeforeCost     float64        `json:"before_cost"`
	AfterCost      float64        `json:"after_cost"`
	DiffCost       float64        `json:"diff_cost"`
	Resources      []ResourceCost `json:"resources"`
	Unsupported    int            `json:"unsupported"`
	UsageEstimated int            `json:"usage_estimated"`
	UsageUnknown   int            `json:"usage_unknown"`
	HasChanges     bool           `json:"has_changes"`
	Error          string         `json:"error,omitempty"`
}

// ModuleError captures a structured error from a failed module estimation.
type ModuleError struct {
	ModuleID string `json:"module_id"`
	Error    string `json:"error"`
}

// PrefetchDiagnostic captures a non-fatal issue encountered while preparing pricing data.
type PrefetchDiagnostic struct {
	Kind         string `json:"kind"`
	ModuleID     string `json:"module_id"`
	ResourceType string `json:"resource_type"`
	Address      string `json:"address"`
	Detail       string `json:"detail,omitempty"`
}

// EstimateResult contains the full cost estimation result.
type EstimateResult struct {
	Modules          []ModuleCost                `json:"modules"`
	Providers        []string                    `json:"providers,omitempty"`
	TotalBefore      float64                     `json:"total_before"`
	TotalAfter       float64                     `json:"total_after"`
	TotalDiff        float64                     `json:"total_diff"`
	Currency         string                      `json:"currency"`
	GeneratedAt      time.Time                   `json:"generated_at"`
	ProviderMetadata map[string]ProviderMetadata `json:"provider_metadata,omitempty"`
	Errors           []ModuleError               `json:"errors,omitempty"`
	PrefetchWarnings []PrefetchDiagnostic        `json:"prefetch_warnings,omitempty"`
	Unsupported      int                         `json:"unsupported"`
	UsageEstimated   int                         `json:"usage_estimated"`
	UsageUnknown     int                         `json:"usage_unknown"`
}

// ProviderMetadata contains provider-specific estimation metadata.
type ProviderMetadata struct {
	DisplayName string `json:"display_name,omitempty"`
	PriceSource string `json:"price_source,omitempty"`
}
