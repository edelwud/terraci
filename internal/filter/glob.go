// Package filter provides filtering functionality for modules based on glob patterns
package filter

import (
	"path/filepath"
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
)

// GlobFilter filters modules based on glob patterns
type GlobFilter struct {
	// ExcludePatterns are patterns to exclude (e.g., "cdp/*/eu-north-1/*")
	ExcludePatterns []string
	// IncludePatterns are patterns to include (if empty, all are included)
	IncludePatterns []string
}

// NewGlobFilter creates a new filter with the given patterns
func NewGlobFilter(exclude, include []string) *GlobFilter {
	return &GlobFilter{
		ExcludePatterns: exclude,
		IncludePatterns: include,
	}
}

// Match checks if a module ID matches the filter criteria
// Returns true if the module should be included
func (f *GlobFilter) Match(moduleID string) bool {
	// Normalize path separators for matching
	normalizedID := filepath.ToSlash(moduleID)

	// Check exclude patterns first
	for _, pattern := range f.ExcludePatterns {
		normalizedPattern := filepath.ToSlash(pattern)
		if matchPattern(normalizedPattern, normalizedID) {
			return false
		}
		// Also try glob-style matching with **
		if matchGlob(normalizedPattern, normalizedID) {
			return false
		}
	}

	// If no include patterns, include by default
	if len(f.IncludePatterns) == 0 {
		return true
	}

	// Check include patterns
	for _, pattern := range f.IncludePatterns {
		normalizedPattern := filepath.ToSlash(pattern)
		if matchPattern(normalizedPattern, normalizedID) {
			return true
		}
		if matchGlob(normalizedPattern, normalizedID) {
			return true
		}
	}

	return false
}

// matchPattern wraps filepath.Match and returns false on invalid patterns
func matchPattern(pattern, name string) bool {
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false // Invalid pattern treated as no match
	}
	return matched
}

// FilterModules returns modules that match the filter criteria
func (f *GlobFilter) FilterModules(modules []*discovery.Module) []*discovery.Module {
	var result []*discovery.Module

	for _, m := range modules {
		if f.Match(m.ID()) {
			result = append(result, m)
		}
	}

	return result
}

// FilterModuleIDs returns module IDs that match the filter criteria
func (f *GlobFilter) FilterModuleIDs(moduleIDs []string) []string {
	var result []string

	for _, id := range moduleIDs {
		if f.Match(id) {
			result = append(result, id)
		}
	}

	return result
}

// matchGlob provides extended glob matching with ** support
func matchGlob(pattern, path string) bool {
	// Handle ** pattern
	if strings.Contains(pattern, "**") {
		return matchDoubleStarGlob(pattern, path)
	}

	// Fall back to standard filepath.Match
	return matchPattern(pattern, path)
}

// matchDoubleStarGlob handles ** patterns that match any number of path segments
func matchDoubleStarGlob(pattern, path string) bool {
	// Split pattern by **
	parts := strings.Split(pattern, "**")

	if len(parts) == 1 {
		// No ** in pattern
		return matchPattern(pattern, path)
	}

	// For pattern like "a/**/b", parts = ["a/", "/b"]
	// Match prefix
	prefix := parts[0]
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/")
		if !strings.HasPrefix(path, prefix) && !matchPrefix(prefix, path) {
			return false
		}
		// Remove matched prefix
		path = strings.TrimPrefix(path, prefix)
		path = strings.TrimPrefix(path, "/")
	}

	// Match suffix
	suffix := parts[len(parts)-1]
	if suffix != "" {
		suffix = strings.TrimPrefix(suffix, "/")
		if !strings.HasSuffix(path, suffix) && !matchSuffix(suffix, path) {
			return false
		}
	}

	// Handle middle parts if any
	if len(parts) > 2 {
		for i := 1; i < len(parts)-1; i++ {
			middle := strings.Trim(parts[i], "/")
			if middle != "" && !strings.Contains(path, middle) {
				return false
			}
		}
	}

	return true
}

// matchPrefix matches a glob prefix against a path
func matchPrefix(prefix, path string) bool {
	prefixParts := strings.Split(prefix, "/")
	pathParts := strings.Split(path, "/")

	if len(prefixParts) > len(pathParts) {
		return false
	}

	for i, pp := range prefixParts {
		if !matchPattern(pp, pathParts[i]) {
			return false
		}
	}

	return true
}

// matchSuffix matches a glob suffix against a path
func matchSuffix(suffix, path string) bool {
	suffixParts := strings.Split(suffix, "/")
	pathParts := strings.Split(path, "/")

	if len(suffixParts) > len(pathParts) {
		return false
	}

	offset := len(pathParts) - len(suffixParts)
	for i, sp := range suffixParts {
		if !matchPattern(sp, pathParts[offset+i]) {
			return false
		}
	}

	return true
}

// ServiceFilter filters modules by service
type ServiceFilter struct {
	Services []string
}

// Match returns true if the module belongs to one of the specified services
func (f *ServiceFilter) Match(module *discovery.Module) bool {
	if len(f.Services) == 0 {
		return true
	}
	for _, s := range f.Services {
		if module.Service == s {
			return true
		}
	}
	return false
}

// EnvironmentFilter filters modules by environment
type EnvironmentFilter struct {
	Environments []string
}

// Match returns true if the module belongs to one of the specified environments
func (f *EnvironmentFilter) Match(module *discovery.Module) bool {
	if len(f.Environments) == 0 {
		return true
	}
	for _, e := range f.Environments {
		if module.Environment == e {
			return true
		}
	}
	return false
}

// RegionFilter filters modules by region
type RegionFilter struct {
	Regions []string
}

// Match returns true if the module is in one of the specified regions
func (f *RegionFilter) Match(module *discovery.Module) bool {
	if len(f.Regions) == 0 {
		return true
	}
	for _, r := range f.Regions {
		if module.Region == r {
			return true
		}
	}
	return false
}

// CompositeFilter combines multiple filters with AND logic
type CompositeFilter struct {
	filters []ModuleFilter
}

// ModuleFilter is an interface for module filters
type ModuleFilter interface {
	Match(module *discovery.Module) bool
}

// NewCompositeFilter creates a composite filter
func NewCompositeFilter(filters ...ModuleFilter) *CompositeFilter {
	return &CompositeFilter{filters: filters}
}

// Match returns true if all filters match
func (f *CompositeFilter) Match(module *discovery.Module) bool {
	for _, filter := range f.filters {
		if !filter.Match(module) {
			return false
		}
	}
	return true
}

// FilterModules applies the composite filter to a list of modules
func (f *CompositeFilter) FilterModules(modules []*discovery.Module) []*discovery.Module {
	var result []*discovery.Module
	for _, m := range modules {
		if f.Match(m) {
			result = append(result, m)
		}
	}
	return result
}

// GlobModuleFilter wraps GlobFilter to implement ModuleFilter interface
type GlobModuleFilter struct {
	*GlobFilter
}

// Match implements ModuleFilter
func (f *GlobModuleFilter) Match(module *discovery.Module) bool {
	return f.GlobFilter.Match(module.ID())
}
