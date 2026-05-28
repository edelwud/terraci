// Package costutil provides attribute extraction and cost calculation utilities
// shared across cloud resource implementations.
package costutil

import "github.com/edelwud/terraci/plugins/cost/internal/resourcedef"

// GetStringAttr extracts a string attribute from resource attributes.
func GetStringAttr(attrs resourcedef.RawAttrs, key string) string {
	return attrs.String(key)
}

// GetFloatAttr extracts a float64 attribute from resource attributes.
func GetFloatAttr(attrs resourcedef.RawAttrs, key string) float64 {
	return attrs.Float(key)
}

// GetIntAttr extracts an int attribute from resource attributes.
func GetIntAttr(attrs resourcedef.RawAttrs, key string) int {
	return attrs.Int(key)
}

// GetBoolAttr extracts a bool attribute from resource attributes.
func GetBoolAttr(attrs resourcedef.RawAttrs, key string) bool {
	return attrs.Bool(key)
}

// GetStringSliceAttr extracts a string slice from resource attributes.
func GetStringSliceAttr(attrs resourcedef.RawAttrs, key string) []string {
	return attrs.StringSlice(key)
}

// GetFirstObjectAttr extracts the first object from a Terraform list block.
func GetFirstObjectAttr(attrs resourcedef.RawAttrs, key string) resourcedef.RawAttrs {
	return attrs.FirstObject(key)
}
