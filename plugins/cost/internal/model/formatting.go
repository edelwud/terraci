package model

const (
	thousandThreshold = 1000
	roundingOffset    = 0.5
	digitsPerGroup    = 3
)

// FormatCost formats a cost value as a string with currency.
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
	if cost >= thousandThreshold {
		return formatWithCommas(cost)
	}
	if cost >= 1 {
		return trimTrailingZeros(cost, 2)
	}
	return trimTrailingZeros(cost, 4)
}

func formatWithCommas(cost float64) string {
	s := trimTrailingZeros(cost, 2)
	parts := splitDecimal(s)
	intPart := parts[0]
	decPart := parts[1]

	numCommas := (len(intPart) - 1) / digitsPerGroup
	result := make([]byte, 0, len(intPart)+numCommas)
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%digitsPerGroup == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c)) //nolint:gosec
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
	format := "%." + string(rune('0'+precision)) + "f" //nolint:gosec
	s := sprintf(format, cost)
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
	neg := f < 0
	if neg {
		f = -f
	}

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

	end := len(s)
	for end > decIdx+1 && s[end-1] == '0' {
		end--
	}
	if end == decIdx+1 {
		end = decIdx
	}
	return s[:end]
}
