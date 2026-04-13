package awskit

import (
	"strconv"
	"strings"
)

// ParseGiB extracts the numeric GiB value from AWS pricing attribute strings
// like "13.07 GiB" or "75 GiB NVMe SSD". It returns 0 when parsing fails.
func ParseGiB(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "none") || s == "0" {
		return 0
	}

	parts := strings.Fields(s)
	if len(parts) == 0 {
		return 0
	}

	v, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	return v
}
