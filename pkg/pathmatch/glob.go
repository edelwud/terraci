// Package pathmatch provides slash-separated path matching helpers.
package pathmatch

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateGlob validates a slash-separated glob pattern. ** is valid only as
// a whole segment and matches zero or more path segments.
func ValidateGlob(pattern string) error {
	for _, segment := range splitPath(pattern) {
		if segment == "**" {
			continue
		}
		if strings.Contains(segment, "**") {
			return fmt.Errorf("invalid ** segment %q", segment)
		}
		if _, err := filepath.Match(segment, ""); err != nil {
			return err
		}
	}
	return nil
}

// MatchGlob reports whether path matches pattern. It returns malformed glob
// errors instead of silently treating them as non-matches.
func MatchGlob(pattern, path string) (bool, error) {
	if err := ValidateGlob(pattern); err != nil {
		return false, err
	}
	return matchSegments(splitPath(pattern), splitPath(path))
}

func splitPath(value string) []string {
	value = filepath.ToSlash(value)
	value = strings.Trim(value, "/")
	if value == "" {
		return nil
	}
	return strings.Split(value, "/")
}

func matchSegments(pattern, path []string) (bool, error) {
	if len(pattern) == 0 {
		return len(path) == 0, nil
	}

	if pattern[0] == "**" {
		for len(pattern) > 1 && pattern[1] == "**" {
			pattern = pattern[1:]
		}
		if len(pattern) == 1 {
			return true, nil
		}
		for i := 0; i <= len(path); i++ {
			matched, err := matchSegments(pattern[1:], path[i:])
			if err != nil || matched {
				return matched, err
			}
		}
		return false, nil
	}

	if len(path) == 0 {
		return false, nil
	}

	matched, err := filepath.Match(pattern[0], path[0])
	if err != nil || !matched {
		return matched, err
	}
	return matchSegments(pattern[1:], path[1:])
}
