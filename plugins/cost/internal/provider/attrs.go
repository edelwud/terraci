package provider

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
func GetFloatAttr(attrs map[string]any, key string) float64 {
	if v, ok := attrs[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0
}

// GetIntAttr extracts an int attribute from a resource attributes map.
func GetIntAttr(attrs map[string]any, key string) int {
	if v, ok := attrs[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case int64:
			return int(val)
		}
	}
	return 0
}

// GetBoolAttr extracts a bool attribute from a resource attributes map.
func GetBoolAttr(attrs map[string]any, key string) bool {
	if v, ok := attrs[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
