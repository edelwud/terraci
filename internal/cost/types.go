// Package cost provides AWS cost estimation for Terraform plans
package cost

import "time"

// Formatting constants
const (
	thousandThreshold = 1000
	roundingOffset    = 0.5
	digitsPerGroup    = 3
)

// ResourceCost represents the estimated cost of a single resource
type ResourceCost struct {
	Address       string  `json:"address"`        // Terraform resource address
	Type          string  `json:"type"`           // Terraform resource type (aws_instance)
	Name          string  `json:"name"`           // Resource name
	Region        string  `json:"region"`         // AWS region
	MonthlyCost   float64 `json:"monthly_cost"`   // Monthly cost in USD
	HourlyCost    float64 `json:"hourly_cost"`    // Hourly cost in USD
	PriceSource   string  `json:"price_source"`   // Source of pricing (aws-bulk-api, cached)
	Unsupported   bool    `json:"unsupported"`    // True if resource type not supported
	UnsupportedBy string  `json:"unsupported_by"` // Reason for unsupported
}

// ModuleCost represents the total cost estimate for a terraform module
type ModuleCost struct {
	ModuleID    string         `json:"module_id"`   // Module identifier
	ModulePath  string         `json:"module_path"` // Path to module
	Region      string         `json:"region"`      // Primary region
	BeforeCost  float64        `json:"before_cost"` // Monthly cost before changes
	AfterCost   float64        `json:"after_cost"`  // Monthly cost after changes
	DiffCost    float64        `json:"diff_cost"`   // Cost difference (after - before)
	Resources   []ResourceCost `json:"resources"`   // Individual resource costs
	Unsupported int            `json:"unsupported"` // Count of unsupported resources
	HasChanges  bool           `json:"has_changes"` // True if costs changed
	Error       string         `json:"error,omitempty"`
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
		result = append(result, byte(c))
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
	format := "%." + string(rune('0'+precision)) + "f"
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
	for i := 0; i < prec; i++ {
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
