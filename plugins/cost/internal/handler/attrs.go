package handler

import (
	"encoding/json"
	"strconv"
)

// GetStringAttr extracts a string attribute from a resource attributes map.
func GetStringAttr(attrs map[string]any, key string) string {
	if v, ok := attrs[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetFloatAttr extracts a float64 attribute from a resource attributes map.
// Handles float64, int, int64, json.Number, and string values.
func GetFloatAttr(attrs map[string]any, key string) float64 {
	if v, ok := attrs[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case int64:
			return float64(val)
		case json.Number:
			if f, err := val.Float64(); err == nil {
				return f
			}
		case string:
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return f
			}
		}
	}
	return 0
}

// GetIntAttr extracts an int attribute from a resource attributes map.
// Handles float64, int, int64, json.Number, and string values.
// float64 values are truncated (not rounded) to int.
func GetIntAttr(attrs map[string]any, key string) int {
	if v, ok := attrs[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case int64:
			return int(val)
		case json.Number:
			if i, err := val.Int64(); err == nil {
				return int(i)
			}
		case string:
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
	}
	return 0
}

// GetBoolAttr extracts a bool attribute from a resource attributes map.
func GetBoolAttr(attrs map[string]any, key string) bool {
	if v, ok := attrs[key]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case string:
			if b, err := strconv.ParseBool(val); err == nil {
				return b
			}
		case int:
			return val != 0
		case int64:
			return val != 0
		case float64:
			return val != 0
		case json.Number:
			if i, err := val.Int64(); err == nil {
				return i != 0
			}
			if f, err := val.Float64(); err == nil {
				return f != 0
			}
		}
	}
	return false
}

// GetStringSliceAttr extracts a string slice from a Terraform attributes map.
func GetStringSliceAttr(attrs map[string]any, key string) []string {
	v, ok := attrs[key]
	if !ok {
		return nil
	}

	switch val := v.(type) {
	case []string:
		return append([]string(nil), val...)
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// GetFirstObjectAttr extracts the first object from a Terraform list block.
func GetFirstObjectAttr(attrs map[string]any, key string) map[string]any {
	v, ok := attrs[key]
	if !ok {
		return nil
	}

	switch val := v.(type) {
	case []map[string]any:
		if len(val) == 0 {
			return nil
		}
		return val[0]
	case []any:
		if len(val) == 0 {
			return nil
		}
		if m, ok := val[0].(map[string]any); ok {
			return m
		}
		return nil
	case map[string]any:
		return val
	default:
		return nil
	}
}
