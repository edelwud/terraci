// Package filter provides filtering functionality for modules based on glob
// patterns and segment values. The public API surface is intentionally tiny:
//
//   - Options describes which filters to apply
//   - Apply runs the filters against a module slice
//   - Flags collects filter values from CLI flags
//   - ParseSegmentFilters parses --filter key=value strings
//
// Concrete filter types (glob, segment, composite) are package-internal.
package filter

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
)

// moduleFilter is the internal predicate over modules that all concrete
// filters in this package satisfy.
type moduleFilter interface {
	match(module *discovery.Module) bool
}

// globFilter filters modules based on include/exclude glob patterns.
type globFilter struct {
	excludes []string
	includes []string
}

func (f globFilter) matchID(moduleID string) bool {
	id := filepath.ToSlash(moduleID)

	for _, pattern := range f.excludes {
		if matchGlob(filepath.ToSlash(pattern), id) {
			return false
		}
	}

	if len(f.includes) == 0 {
		return true
	}

	for _, pattern := range f.includes {
		if matchGlob(filepath.ToSlash(pattern), id) {
			return true
		}
	}

	return false
}

func (f globFilter) match(module *discovery.Module) bool {
	return f.matchID(module.ID())
}

// segmentFilter filters modules by a named segment value.
type segmentFilter struct {
	segment string
	values  []string
}

func (f segmentFilter) match(module *discovery.Module) bool {
	if len(f.values) == 0 {
		return true
	}
	return slices.Contains(f.values, module.Get(f.segment))
}

// Options contains all filter parameters.
type Options struct {
	Excludes []string            // glob patterns to exclude
	Includes []string            // glob patterns to include (empty = all)
	Segments map[string][]string // segment name → allowed values (e.g. "service" → ["platform"])
}

// Apply applies all configured filters to modules. Returns the input slice
// unchanged when no filter criteria are set.
func Apply(modules []*discovery.Module, opts Options) []*discovery.Module {
	filters := opts.filters()
	if len(filters) == 0 {
		return modules
	}

	result := make([]*discovery.Module, 0, len(modules))
	for _, m := range modules {
		if matchAll(filters, m) {
			result = append(result, m)
		}
	}
	return result
}

func (o Options) filters() []moduleFilter {
	var filters []moduleFilter
	if len(o.Excludes) > 0 || len(o.Includes) > 0 {
		filters = append(filters, globFilter{excludes: o.Excludes, includes: o.Includes})
	}
	for segment, values := range o.Segments {
		if len(values) > 0 {
			filters = append(filters, segmentFilter{segment: segment, values: values})
		}
	}
	return filters
}

func matchAll(filters []moduleFilter, module *discovery.Module) bool {
	for _, f := range filters {
		if !f.match(module) {
			return false
		}
	}
	return true
}

// ParseSegmentFilters parses "key=value" strings into a segment filter map.
func ParseSegmentFilters(args []string) map[string][]string {
	segments := make(map[string][]string)
	for _, arg := range args {
		if k, v, ok := strings.Cut(arg, "="); ok && k != "" {
			segments[k] = append(segments[k], v)
		}
	}
	return segments
}

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

	if prefix := strings.TrimSuffix(parts[0], "/"); prefix != "" {
		if !strings.HasPrefix(path, prefix) && !matchSegments(prefix, path, true) {
			return false
		}
		path = strings.TrimPrefix(strings.TrimPrefix(path, prefix), "/")
	}

	if suffix := strings.TrimPrefix(parts[len(parts)-1], "/"); suffix != "" {
		if !strings.HasSuffix(path, suffix) && !matchSegments(suffix, path, false) {
			return false
		}
	}

	for i := 1; i < len(parts)-1; i++ {
		if middle := strings.Trim(parts[i], "/"); middle != "" && !strings.Contains(path, middle) {
			return false
		}
	}

	return true
}

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
