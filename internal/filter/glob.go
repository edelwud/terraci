// Package filter provides filtering functionality for modules based on glob patterns
// and segment values.
package filter

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
)

// ModuleFilter is an interface for module filters.
type ModuleFilter interface {
	Match(module *discovery.Module) bool
}

// --- Glob filter ---

// GlobFilter filters modules based on include/exclude glob patterns.
type GlobFilter struct {
	ExcludePatterns []string
	IncludePatterns []string
}

// NewGlobFilter creates a new filter with the given patterns.
func NewGlobFilter(exclude, include []string) *GlobFilter {
	return &GlobFilter{
		ExcludePatterns: exclude,
		IncludePatterns: include,
	}
}

// Match checks if a module ID matches the filter criteria.
func (f *GlobFilter) Match(moduleID string) bool {
	id := filepath.ToSlash(moduleID)

	for _, pattern := range f.ExcludePatterns {
		if matchGlob(filepath.ToSlash(pattern), id) {
			return false
		}
	}

	if len(f.IncludePatterns) == 0 {
		return true
	}

	for _, pattern := range f.IncludePatterns {
		if matchGlob(filepath.ToSlash(pattern), id) {
			return true
		}
	}

	return false
}

// MatchModule implements ModuleFilter.
func (f *GlobFilter) MatchModule(module *discovery.Module) bool {
	return f.Match(module.ID())
}

// FilterModules returns modules that match the filter criteria.
func (f *GlobFilter) FilterModules(modules []*discovery.Module) []*discovery.Module {
	var result []*discovery.Module
	for _, m := range modules {
		if f.Match(m.ID()) {
			result = append(result, m)
		}
	}
	return result
}

// FilterModuleIDs returns module IDs that match the filter criteria.
func (f *GlobFilter) FilterModuleIDs(moduleIDs []string) []string {
	var result []string
	for _, id := range moduleIDs {
		if f.Match(id) {
			result = append(result, id)
		}
	}
	return result
}

// --- Segment filter (replaces ServiceFilter, EnvironmentFilter, RegionFilter) ---

// SegmentFilter filters modules by a named segment value.
type SegmentFilter struct {
	Segment string   // segment name (e.g. "service", "environment", "region")
	Values  []string // allowed values
}

// Match implements ModuleFilter.
func (f *SegmentFilter) Match(module *discovery.Module) bool {
	if len(f.Values) == 0 {
		return true
	}
	return slices.Contains(f.Values, module.Get(f.Segment))
}

// --- Composite filter ---

// CompositeFilter combines multiple filters with AND logic.
type CompositeFilter struct {
	filters []ModuleFilter
}

// NewCompositeFilter creates a composite filter.
func NewCompositeFilter(filters ...ModuleFilter) *CompositeFilter {
	return &CompositeFilter{filters: filters}
}

// Match returns true if all filters match.
func (f *CompositeFilter) Match(module *discovery.Module) bool {
	for _, filter := range f.filters {
		if !filter.Match(module) {
			return false
		}
	}
	return true
}

// FilterModules applies the composite filter to a list of modules.
func (f *CompositeFilter) FilterModules(modules []*discovery.Module) []*discovery.Module {
	var result []*discovery.Module
	for _, m := range modules {
		if f.Match(m) {
			result = append(result, m)
		}
	}
	return result
}

// --- Options and Apply ---

// Options contains all filter parameters.
type Options struct {
	Excludes []string            // glob patterns to exclude
	Includes []string            // glob patterns to include (empty = all)
	Segments map[string][]string // segment name → allowed values (e.g. "service" → ["platform"])
}

// Apply applies all configured filters to modules.
func Apply(modules []*discovery.Module, opts Options) []*discovery.Module {
	var filters []ModuleFilter

	if len(opts.Excludes) > 0 || len(opts.Includes) > 0 {
		filters = append(filters, &globModuleFilter{NewGlobFilter(opts.Excludes, opts.Includes)})
	}

	for segment, values := range opts.Segments {
		if len(values) > 0 {
			filters = append(filters, &SegmentFilter{Segment: segment, Values: values})
		}
	}

	if len(filters) == 0 {
		return modules
	}

	return NewCompositeFilter(filters...).FilterModules(modules)
}

// globModuleFilter adapts GlobFilter to ModuleFilter interface.
type globModuleFilter struct{ *GlobFilter }

func (f *globModuleFilter) Match(module *discovery.Module) bool {
	return f.GlobFilter.Match(module.ID())
}

// --- Glob matching internals ---

// matchGlob provides extended glob matching with ** support.
func matchGlob(pattern, path string) bool {
	if strings.Contains(pattern, "**") {
		return matchDoubleStarGlob(pattern, path)
	}
	matched, _ := filepath.Match(pattern, path) //nolint:errcheck
	return matched
}

func matchDoubleStarGlob(pattern, path string) bool {
	parts := strings.Split(pattern, "**")
	if len(parts) == 1 {
		matched, _ := filepath.Match(pattern, path) //nolint:errcheck
		return matched
	}

	// Match prefix
	if prefix := strings.TrimSuffix(parts[0], "/"); prefix != "" {
		if !strings.HasPrefix(path, prefix) && !matchSegments(prefix, path, true) {
			return false
		}
		path = strings.TrimPrefix(strings.TrimPrefix(path, prefix), "/")
	}

	// Match suffix
	if suffix := strings.TrimPrefix(parts[len(parts)-1], "/"); suffix != "" {
		if !strings.HasSuffix(path, suffix) && !matchSegments(suffix, path, false) {
			return false
		}
	}

	// Match middle parts
	for i := 1; i < len(parts)-1; i++ {
		if middle := strings.Trim(parts[i], "/"); middle != "" && !strings.Contains(path, middle) {
			return false
		}
	}

	return true
}

// matchSegments matches glob segments against path segments (prefix or suffix).
func matchSegments(pattern, path string, prefix bool) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) > len(pathParts) {
		return false
	}

	offset := 0
	if !prefix {
		offset = len(pathParts) - len(patternParts)
	}

	for i, pp := range patternParts {
		matched, _ := filepath.Match(pp, pathParts[offset+i]) //nolint:errcheck
		if !matched {
			return false
		}
	}
	return true
}
