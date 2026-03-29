package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// PatternSegments represents the ordered list of placeholder names from a structure pattern.
// For example, "{service}/{environment}/{region}/{module}" yields ["service", "environment", "region", "module"].
type PatternSegments []string

// ParsePattern parses a structure pattern string into ordered segment names.
// Each segment must be a {name} placeholder. Returns an error if the pattern is invalid.
func ParsePattern(pattern string) (PatternSegments, error) {
	if pattern == "" {
		return nil, errors.New("pattern is empty")
	}

	parts := strings.Split(pattern, "/")
	segments := make(PatternSegments, 0, len(parts))
	seen := make(map[string]bool, len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "{") || !strings.HasSuffix(part, "}") {
			return nil, fmt.Errorf("segment %d (%q) must be a {name} placeholder", i, part)
		}

		name := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
		if name == "" {
			return nil, fmt.Errorf("segment %d has empty placeholder name", i)
		}

		if seen[name] {
			return nil, fmt.Errorf("duplicate placeholder {%s}", name)
		}
		seen[name] = true
		segments = append(segments, name)
	}

	if len(segments) == 0 {
		return nil, errors.New("pattern must have at least one segment")
	}

	return segments, nil
}

// Contains returns true if the segments include the given name.
func (ps PatternSegments) Contains(name string) bool {
	return slices.Contains(ps, name)
}

// IndexOf returns the position of the given name, or -1 if not found.
func (ps PatternSegments) IndexOf(name string) int {
	for i, s := range ps {
		if s == name {
			return i
		}
	}
	return -1
}

// LeafName returns the last segment name (the "module" equivalent).
func (ps PatternSegments) LeafName() string {
	if len(ps) == 0 {
		return ""
	}
	return ps[len(ps)-1]
}

// ContextNames returns all segment names except the last (the "context" prefix).
func (ps PatternSegments) ContextNames() []string {
	if len(ps) <= 1 {
		return nil
	}
	return ps[:len(ps)-1]
}
