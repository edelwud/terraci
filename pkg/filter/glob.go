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
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pathmatch"
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

// Matcher is a validated filter predicate. Build one with Options.Compile()
// to amortize filter construction across many module checks — callers that
// invoke Apply on slice-of-1 inside a loop can hoist Compile() out and reuse
// the Matcher in the inner loop.
type Matcher struct {
	filters []moduleFilter
}

// Empty reports whether the matcher has no active filters; callers can
// short-circuit and accept every module.
func (m Matcher) Empty() bool { return len(m.filters) == 0 }

// Matches reports whether a module passes all configured filters.
func (m Matcher) Matches(module *discovery.Module) bool {
	if len(m.filters) == 0 {
		return true
	}
	return matchAll(m.filters, module)
}

// Validate checks filter options before they are compiled.
func (o Options) Validate() error {
	for i, pattern := range o.Excludes {
		if err := pathmatch.ValidateGlob(pattern); err != nil {
			return fmt.Errorf("exclude[%d]: %w", i, err)
		}
	}
	for i, pattern := range o.Includes {
		if err := pathmatch.ValidateGlob(pattern); err != nil {
			return fmt.Errorf("include[%d]: %w", i, err)
		}
	}
	return nil
}

// Compile builds a reusable Matcher from Options. Equivalent to extracting
// the internal filter list once instead of rebuilding it per Apply call.
func (o Options) Compile() (Matcher, error) {
	if err := o.Validate(); err != nil {
		return Matcher{}, err
	}

	var filters []moduleFilter
	if len(o.Excludes) > 0 || len(o.Includes) > 0 {
		filters = append(filters, globFilter{excludes: o.Excludes, includes: o.Includes})
	}
	for segment, values := range o.Segments {
		if len(values) > 0 {
			filters = append(filters, segmentFilter{segment: segment, values: values})
		}
	}
	return Matcher{filters: filters}, nil
}

// Apply applies all configured filters to modules. Returns the input slice
// unchanged when no filter criteria are set.
func Apply(modules []*discovery.Module, opts Options) ([]*discovery.Module, error) {
	matcher, err := opts.Compile()
	if err != nil {
		return nil, err
	}
	if matcher.Empty() {
		return modules, nil
	}

	result := make([]*discovery.Module, 0, len(modules))
	for _, m := range modules {
		if matcher.Matches(m) {
			result = append(result, m)
		}
	}
	return result, nil
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
	matched, err := pathmatch.MatchGlob(pattern, path)
	return err == nil && matched
}
