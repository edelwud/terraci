// Package cost provides AWS cost estimation for Terraform plans
package costengine

import (
	"errors"
	"fmt"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
)

// Formatting constants
const (
	thousandThreshold = 1000
	roundingOffset    = 0.5
	digitsPerGroup    = 3
)

// ResourceCost represents the estimated cost of a single resource
type ResourceCost struct {
	Provider          string            `json:"provider,omitempty"`            // Cloud provider id (aws, gcp, azure)
	Address           string            `json:"address"`                       // Terraform resource address
	ModuleAddr        string            `json:"module_addr,omitempty"`         // Terraform module address (e.g., "module.vpc")
	Type              string            `json:"type"`                          // Terraform resource type (aws_instance)
	Name              string            `json:"name"`                          // Resource name
	Region            string            `json:"region"`                        // Cloud region
	MonthlyCost       float64           `json:"monthly_cost"`                  // After-state monthly cost in USD
	HourlyCost        float64           `json:"hourly_cost"`                   // After-state hourly cost in USD
	BeforeMonthlyCost float64           `json:"before_monthly_cost,omitempty"` // Before-state monthly cost (update/replace)
	BeforeHourlyCost  float64           `json:"before_hourly_cost,omitempty"`  // Before-state hourly cost (update/replace)
	PriceSource       string            `json:"price_source"`                  // Source of pricing (provider API, fixed, usage-based)
	ErrorKind         CostErrorKind     `json:"error_kind,omitempty"`          // Classification of estimation error
	ErrorDetail       string            `json:"error_detail,omitempty"`        // Human-readable error detail
	Details           map[string]string `json:"details,omitempty"`             // Resource-specific info (instance_type, nodes, disk, etc.)
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
// Children contains nested submodules (e.g., module.eks contains module.eks.module.node_group).
// MonthlyCost reflects only direct resources; use TotalCost() for recursive totals.
type SubmoduleCost struct {
	ModuleAddr  string          `json:"module_addr"`         // e.g., "module.runner" or "" for root
	MonthlyCost float64         `json:"monthly_cost"`        // Direct resources only
	Resources   []ResourceCost  `json:"resources,omitempty"` // Direct resources only
	Children    []SubmoduleCost `json:"children,omitempty"`  // Nested submodules
}

// TotalCost returns MonthlyCost including all nested children recursively.
func (s *SubmoduleCost) TotalCost() float64 {
	total := s.MonthlyCost
	for i := range s.Children {
		total += s.Children[i].TotalCost()
	}
	return total
}

// ModuleCost represents the total cost estimate for a terraform module
type ModuleCost struct {
	ModuleID    string          `json:"module_id"`            // Module identifier
	ModulePath  string          `json:"module_path"`          // Path to module
	Region      string          `json:"region"`               // Primary region
	Provider    string          `json:"provider,omitempty"`   // Single provider id when the module is homogeneous
	Providers   []string        `json:"providers,omitempty"`  // All providers encountered in the module
	BeforeCost  float64         `json:"before_cost"`          // Monthly cost before changes
	AfterCost   float64         `json:"after_cost"`           // Monthly cost after changes
	DiffCost    float64         `json:"diff_cost"`            // Cost difference (after - before)
	Resources   []ResourceCost  `json:"resources"`            // Flat list of all resource costs
	Submodules  []SubmoduleCost `json:"submodules,omitempty"` // Grouped by Terraform module address
	Unsupported int             `json:"unsupported"`          // Count of unsupported resources
	HasChanges  bool            `json:"has_changes"`          // True if costs changed
	Error       string          `json:"error,omitempty"`
}

// ModuleError captures a structured error from a failed module estimation.
type ModuleError struct {
	ModuleID string `json:"module_id"`
	Error    string `json:"error"`
}

// EstimateResult contains the full cost estimation result
type EstimateResult struct {
	Modules          []ModuleCost                `json:"modules"`
	Providers        []string                    `json:"providers,omitempty"`
	TotalBefore      float64                     `json:"total_before"`
	TotalAfter       float64                     `json:"total_after"`
	TotalDiff        float64                     `json:"total_diff"`
	Currency         string                      `json:"currency"` // USD for the built-in AWS provider
	GeneratedAt      time.Time                   `json:"generated_at"`
	ProviderMetadata map[string]ProviderMetadata `json:"provider_metadata,omitempty"`
	Errors           []ModuleError               `json:"errors,omitempty"` // Modules that failed estimation
}

// ProviderMetadata contains provider-specific estimation metadata.
type ProviderMetadata struct {
	DisplayName string `json:"display_name,omitempty"`
	PriceSource string `json:"price_source,omitempty"`
}

// FormatCost formats a cost value as a string with currency
func FormatCost(cost float64) string {
	if cost == 0 {
		return "$0"
	}
	if cost < 0.01 && cost > -0.01 && cost != 0 {
		return "<$0.01"
	}
	if cost < 0 {
		return "-$" + formatPositive(-cost)
	}
	return "$" + formatPositive(cost)
}

// FormatCostDiff formats a cost difference with +/- prefix
func FormatCostDiff(diff float64) string {
	if diff == 0 {
		return "$0"
	}
	if diff > 0 {
		return "+" + FormatCost(diff)
	}
	return FormatCost(diff) // Already includes minus
}

func formatPositive(cost float64) string {
	if cost >= thousandThreshold {
		return formatWithCommas(cost)
	}
	if cost >= 1 {
		return trimTrailingZeros(cost, 2)
	}
	return trimTrailingZeros(cost, 4)
}

func formatWithCommas(cost float64) string {
	// Simple comma formatting for thousands
	s := trimTrailingZeros(cost, 2)
	parts := splitDecimal(s)
	intPart := parts[0]
	decPart := parts[1]

	// Add commas to integer part
	// Pre-allocate result buffer: original length + commas (1 per 3 digits after first group)
	numCommas := (len(intPart) - 1) / digitsPerGroup
	result := make([]byte, 0, len(intPart)+numCommas)
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%digitsPerGroup == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c)) //nolint:gosec // c is always an ASCII digit (0-9)
	}

	if decPart != "" {
		return string(result) + "." + decPart
	}
	return string(result)
}

func splitDecimal(s string) [2]string {
	for i, c := range s {
		if c == '.' {
			return [2]string{s[:i], s[i+1:]}
		}
	}
	return [2]string{s, ""}
}

func trimTrailingZeros(cost float64, precision int) string {
	format := "%." + string(rune('0'+precision)) + "f" //nolint:gosec // precision is always a small non-negative int
	s := sprintf(format, cost)
	// Trim trailing zeros after decimal
	if hasDecimal(s) {
		s = trimZeros(s)
	}
	return s
}

func sprintf(format string, cost float64) string {
	switch format {
	case "%.2f":
		return sprintfFloat(cost, 2)
	case "%.4f":
		return sprintfFloat(cost, 4)
	default:
		return sprintfFloat(cost, 2)
	}
}

func sprintfFloat(f float64, prec int) string {
	// Simple float formatting without fmt dependency
	neg := f < 0
	if neg {
		f = -f
	}

	// Scale by precision
	scale := 1.0
	for range prec {
		scale *= 10
	}
	rounded := int64(f*scale + roundingOffset)

	intPart := rounded / int64(scale)
	fracPart := rounded % int64(scale)

	var result string
	if neg {
		result = "-"
	}
	result += itoa(intPart)

	if prec > 0 {
		result += "."
		fracStr := itoa(fracPart)
		for len(fracStr) < prec {
			fracStr = "0" + fracStr
		}
		result += fracStr
	}

	return result
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func hasDecimal(s string) bool {
	for _, c := range s {
		if c == '.' {
			return true
		}
	}
	return false
}

func trimZeros(s string) string {
	// Find decimal point
	decIdx := -1
	for i, c := range s {
		if c == '.' {
			decIdx = i
			break
		}
	}
	if decIdx == -1 {
		return s
	}

	// Trim trailing zeros
	end := len(s)
	for end > decIdx+1 && s[end-1] == '0' {
		end--
	}
	// Remove decimal if no fraction
	if end == decIdx+1 {
		end = decIdx
	}
	return s[:end]
}

// CostConfig defines configuration for cost estimation.
type CostConfig struct {
	// Enabled is an internal shorthand used by tests; user-facing config should use providers.<name>.enabled.
	Enabled bool `yaml:"-" json:"-"`

	// CacheDir is the directory to cache AWS pricing data
	// If empty, uses ~/.terraci/pricing
	CacheDir string `yaml:"cache_dir,omitempty" json:"cache_dir,omitempty" jsonschema:"description=Directory to cache AWS pricing data"`

	// CacheTTL is how long cached pricing data is valid (e.g., '24h', '7d')
	// Default: 24h
	CacheTTL string `yaml:"cache_ttl,omitempty" json:"cache_ttl,omitempty" jsonschema:"description=How long cached pricing is valid (e.g. 24h),default=24h"`

	// Providers contains provider-specific settings.
	Providers CostProvidersConfig `yaml:"providers" json:"providers"`

	// LegacyEnabled exists only to surface a clear migration error.
	LegacyEnabled *bool `yaml:"enabled,omitempty" json:"-"`
}

// CostProvidersConfig contains built-in provider configs.
type CostProvidersConfig struct {
	AWS *ProviderConfig `yaml:"aws,omitempty" json:"aws,omitempty"`
}

// ProviderConfig contains provider activation state.
type ProviderConfig struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable this cloud provider,default=false"`
}

// EnabledProviderIDs returns all enabled cloud providers.
func (c *CostConfig) EnabledProviderIDs() []string {
	if c == nil {
		return nil
	}

	var providers []string
	if c.Providers.AWS != nil && c.Providers.AWS.Enabled {
		providers = append(providers, awskit.ProviderID)
	}
	if c.Enabled {
		providers = append(providers, awskit.ProviderID)
	}

	return providers
}

// HasEnabledProviders returns true when at least one provider is enabled.
func (c *CostConfig) HasEnabledProviders() bool {
	return len(c.EnabledProviderIDs()) > 0
}

// Validate checks if the CostConfig values are valid.
func (c *CostConfig) Validate() error {
	if c.LegacyEnabled != nil {
		return errors.New("plugins.cost.enabled is no longer supported; use plugins.cost.providers.aws.enabled")
	}
	if c.CacheTTL != "" {
		if _, err := time.ParseDuration(c.CacheTTL); err != nil {
			return fmt.Errorf("invalid cache_ttl %q: %w", c.CacheTTL, err)
		}
	}
	if !c.HasEnabledProviders() {
		return nil
	}
	return nil
}
