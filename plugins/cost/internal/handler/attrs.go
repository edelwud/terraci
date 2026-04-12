package handler

import "github.com/edelwud/terraci/plugins/cost/internal/costutil"

// GetStringAttr extracts a string attribute from a resource attributes map.
func GetStringAttr(attrs map[string]any, key string) string {
	return costutil.GetStringAttr(attrs, key)
}

// GetFloatAttr extracts a float64 attribute from a resource attributes map.
func GetFloatAttr(attrs map[string]any, key string) float64 {
	return costutil.GetFloatAttr(attrs, key)
}

// GetIntAttr extracts an int attribute from a resource attributes map.
func GetIntAttr(attrs map[string]any, key string) int {
	return costutil.GetIntAttr(attrs, key)
}

// GetBoolAttr extracts a bool attribute from a resource attributes map.
func GetBoolAttr(attrs map[string]any, key string) bool {
	return costutil.GetBoolAttr(attrs, key)
}

// GetStringSliceAttr extracts a string slice from a Terraform attributes map.
func GetStringSliceAttr(attrs map[string]any, key string) []string {
	return costutil.GetStringSliceAttr(attrs, key)
}

// GetFirstObjectAttr extracts the first object from a Terraform list block.
func GetFirstObjectAttr(attrs map[string]any, key string) map[string]any {
	return costutil.GetFirstObjectAttr(attrs, key)
}
