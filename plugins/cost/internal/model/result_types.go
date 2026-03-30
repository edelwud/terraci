package model

import "time"

// ResourceCost represents the estimated cost of a single resource.
type ResourceCost struct {
	Provider          string            `json:"provider,omitempty"`
	Address           string            `json:"address"`
	ModuleAddr        string            `json:"module_addr,omitempty"`
	Type              string            `json:"type"`
	Name              string            `json:"name"`
	Region            string            `json:"region"`
	MonthlyCost       float64           `json:"monthly_cost"`
	HourlyCost        float64           `json:"hourly_cost"`
	BeforeMonthlyCost float64           `json:"before_monthly_cost,omitempty"`
	BeforeHourlyCost  float64           `json:"before_hourly_cost,omitempty"`
	PriceSource       string            `json:"price_source"`
	ErrorKind         CostErrorKind     `json:"error_kind,omitempty"`
	ErrorDetail       string            `json:"error_detail,omitempty"`
	Details           map[string]string `json:"details,omitempty"`
}

// IsUnsupported returns true if the resource cost could not be estimated
// (excluding usage-based resources which are expected to have no static price).
func (r ResourceCost) IsUnsupported() bool {
	return r.ErrorKind != CostErrorNone && r.ErrorKind != CostErrorUsageBased
}

// CostErrorKind classifies the type of cost estimation error.
type CostErrorKind string

const (
	CostErrorNone         CostErrorKind = ""
	CostErrorNoProvider   CostErrorKind = "no_provider"
	CostErrorNoHandler    CostErrorKind = "no_handler"
	CostErrorUsageBased   CostErrorKind = "usage_based"
	CostErrorLookupFailed CostErrorKind = "lookup_failed"
	CostErrorAPIFailure   CostErrorKind = "api_failure"
	CostErrorNoPrice      CostErrorKind = "no_price"
)

// SubmoduleCost groups resource costs by Terraform module address.
type SubmoduleCost struct {
	ModuleAddr  string          `json:"module_addr"`
	MonthlyCost float64         `json:"monthly_cost"`
	Resources   []ResourceCost  `json:"resources,omitempty"`
	Children    []SubmoduleCost `json:"children,omitempty"`
}

// TotalCost returns MonthlyCost including all nested children recursively.
func (s *SubmoduleCost) TotalCost() float64 {
	total := s.MonthlyCost
	for i := range s.Children {
		total += s.Children[i].TotalCost()
	}
	return total
}

// ModuleCost represents the total cost estimate for a terraform module.
type ModuleCost struct {
	ModuleID    string          `json:"module_id"`
	ModulePath  string          `json:"module_path"`
	Region      string          `json:"region"`
	Provider    string          `json:"provider,omitempty"`
	Providers   []string        `json:"providers,omitempty"`
	BeforeCost  float64         `json:"before_cost"`
	AfterCost   float64         `json:"after_cost"`
	DiffCost    float64         `json:"diff_cost"`
	Resources   []ResourceCost  `json:"resources"`
	Submodules  []SubmoduleCost `json:"submodules,omitempty"`
	Unsupported int             `json:"unsupported"`
	HasChanges  bool            `json:"has_changes"`
	Error       string          `json:"error,omitempty"`
}

// ModuleError captures a structured error from a failed module estimation.
type ModuleError struct {
	ModuleID string `json:"module_id"`
	Error    string `json:"error"`
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
}

// ProviderMetadata contains provider-specific estimation metadata.
type ProviderMetadata struct {
	DisplayName string `json:"display_name,omitempty"`
	PriceSource string `json:"price_source,omitempty"`
}
