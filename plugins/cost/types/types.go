// Package cost provides AWS cost estimation for Terraform plans
package types

import "time"

// Formatting constants
const (
	thousandThreshold = 1000
	roundingOffset    = 0.5
	digitsPerGroup    = 3
)

// ResourceCost represents the estimated cost of a single resource
type ResourceCost struct {
	Address           string            `json:"address"`                       // Terraform resource address
	ModuleAddr        string            `json:"module_addr,omitempty"`         // Terraform module address (e.g., "module.vpc")
	Type              string            `json:"type"`                          // Terraform resource type (aws_instance)
	Name              string            `json:"name"`                          // Resource name
	Region            string            `json:"region"`                        // AWS region
	MonthlyCost       float64           `json:"monthly_cost"`                  // After-state monthly cost in USD
	HourlyCost        float64           `json:"hourly_cost"`                   // After-state hourly cost in USD
	BeforeMonthlyCost float64           `json:"before_monthly_cost,omitempty"` // Before-state monthly cost (update/replace)
	BeforeHourlyCost  float64           `json:"before_hourly_cost,omitempty"`  // Before-state hourly cost (update/replace)
	PriceSource       string            `json:"price_source"`                  // Source of pricing (aws-bulk-api, fixed, usage-based)
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
type CostErrorKind string //nolint:revive // name is intentional for clarity outside the package

const (
	CostErrorNone         CostErrorKind = ""
	CostErrorNoHandler    CostErrorKind = "no_handler"
	CostErrorUsageBased   CostErrorKind = "usage_based"
	CostErrorLookupFailed CostErrorKind = "lookup_failed"
	CostErrorAPIFailure   CostErrorKind = "api_failure"
	CostErrorNoPrice      CostErrorKind = "no_price"
)

// SubmoduleCost groups resource costs by Terraform module address.
// Children contains nested submodules (e.g., module.eks contains module.eks.module.node_group).
type SubmoduleCost struct {
	ModuleAddr  string          `json:"module_addr"`         // e.g., "module.runner" or "" for root
	MonthlyCost float64         `json:"monthly_cost"`        // Total including children
	Resources   []ResourceCost  `json:"resources,omitempty"` // Direct resources only
	Children    []SubmoduleCost `json:"children,omitempty"`  // Nested submodules
}

// ModuleCost represents the total cost estimate for a terraform module
type ModuleCost struct {
	ModuleID    string          `json:"module_id"`            // Module identifier
	ModulePath  string          `json:"module_path"`          // Path to module
	Region      string          `json:"region"`               // Primary region
	BeforeCost  float64         `json:"before_cost"`          // Monthly cost before changes
	AfterCost   float64         `json:"after_cost"`           // Monthly cost after changes
	DiffCost    float64         `json:"diff_cost"`            // Cost difference (after - before)
	Resources   []ResourceCost  `json:"resources"`            // Flat list of all resource costs
	Submodules  []SubmoduleCost `json:"submodules,omitempty"` // Grouped by Terraform module address
	Unsupported int             `json:"unsupported"`          // Count of unsupported resources
	HasChanges  bool            `json:"has_changes"`          // True if costs changed
	Error       string          `json:"error,omitempty"`
}

// EstimateResult contains the full cost estimation result
type EstimateResult struct {
	Modules        []ModuleCost `json:"modules"`
	TotalBefore    float64      `json:"total_before"`
	TotalAfter     float64      `json:"total_after"`
	TotalDiff      float64      `json:"total_diff"`
	Currency       string       `json:"currency"` // USD
	GeneratedAt    time.Time    `json:"generated_at"`
	PricingVersion string       `json:"pricing_version"` // AWS pricing version/date
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
