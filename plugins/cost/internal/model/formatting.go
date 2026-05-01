package model

import (
	"fmt"
	"strings"
)

const digitsPerGroup = 3

// FormatCost formats a cost value as a string with currency.
func FormatCost(cost float64) string {
	if cost == 0 {
		return "$0"
	}
	if cost < 0.01 && cost > -0.01 {
		return "<$0.01"
	}
	if cost < 0 {
		return "-$" + formatPositive(-cost)
	}
	return "$" + formatPositive(cost)
}

// FormatCostDiff formats a cost difference with +/- prefix.
func FormatCostDiff(diff float64) string {
	if diff == 0 {
		return "$0"
	}
	if diff > 0 {
		return "+" + FormatCost(diff)
	}
	return FormatCost(diff)
}

func formatPositive(cost float64) string {
	if cost >= 1000 {
		return formatWithCommas(cost)
	}
	if cost >= 1 {
		return trimTrailingZeros(cost, 2)
	}
	return trimTrailingZeros(cost, 4)
}

func formatWithCommas(cost float64) string {
	s := trimTrailingZeros(cost, 2)
	intPart, decPart, _ := strings.Cut(s, ".")

	numCommas := (len(intPart) - 1) / digitsPerGroup
	result := make([]byte, 0, len(intPart)+numCommas)
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%digitsPerGroup == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c)) //nolint:gosec // c is always an ASCII digit [0-9] or separator
	}

	if decPart != "" {
		return string(result) + "." + decPart
	}
	return string(result)
}

func trimTrailingZeros(cost float64, precision int) string {
	s := fmt.Sprintf("%.*f", precision, cost)
	if strings.ContainsRune(s, '.') {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}
