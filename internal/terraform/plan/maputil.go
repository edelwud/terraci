package plan

import (
	"fmt"
	"strconv"
	"strings"
)

// toMap converts an any to map[string]any, returning nil for non-maps.
func toMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

// collectKeys recursively collects all dot-separated keys from a nested map.
func collectKeys(m map[string]any, prefix string, keys map[string]bool) {
	for k, v := range m {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}
		keys[fullKey] = true

		if nested, ok := v.(map[string]any); ok {
			collectKeys(nested, fullKey, keys)
		}
	}
}

// getNestedValue gets a value from a nested map using dot notation.
func getNestedValue(m map[string]any, key string) any {
	if m == nil {
		return nil
	}

	var current any = m
	for part := range strings.SplitSeq(key, ".") {
		cm, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = cm[part]
	}
	return current
}

// getBool gets a boolean value from a nested map.
func getBool(m map[string]any, key string) bool {
	b, _ := getNestedValue(m, key).(bool) //nolint:errcheck
	return b
}

// formatValue formats a value for display.
func formatValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return fmt.Sprintf("%g", val)
	case []any:
		if len(val) == 0 {
			return "[]"
		}
		return fmt.Sprintf("[%d items]", len(val))
	case map[string]any:
		if len(val) == 0 {
			return "{}"
		}
		return fmt.Sprintf("{%d keys}", len(val))
	default:
		return fmt.Sprintf("%v", v)
	}
}

// pathToString converts a path array to a dot-separated string.
func pathToString(paths []any) string {
	parts := make([]string, 0, len(paths))
	for _, p := range paths {
		if s, ok := p.(string); ok {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ".")
}
