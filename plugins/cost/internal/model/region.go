package model

import (
	"path/filepath"
	"strings"
)

// DefaultRegion is used when no region is specified.
const DefaultRegion = "us-east-1"

// DetectRegion extracts region from a module path using configured pattern segments.
func DetectRegion(segments []string, modulePath string) string {
	parts := splitPath(modulePath)
	for i, seg := range segments {
		if seg == "region" && i < len(parts) {
			return parts[i]
		}
	}
	return DefaultRegion
}

// splitPath splits a filepath into its OS-independent path segments.
func splitPath(p string) []string {
	return strings.Split(filepath.ToSlash(filepath.Clean(p)), "/")
}
