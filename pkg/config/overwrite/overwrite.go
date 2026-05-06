// Package overwrite provides small helpers for resolving ordered YAML
// overwrite rules without coupling callers to a specific config shape.
package overwrite

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/pathmatch"
)

// Matcher decides whether an overwrite applies to the target.
type Matcher[O, T any] func(overwrite *O, target T) (bool, error)

// Applier mutates an already-created effective config with one matching
// overwrite. Field-level merge semantics stay with each owning package.
type Applier[E, O any] func(effective *E, overwrite *O)

// ApplyMatching applies every matching overwrite in declaration order.
func ApplyMatching[E, O, T any](
	effective *E,
	target T,
	overwrites []O,
	match Matcher[O, T],
	apply Applier[E, O],
) error {
	if effective == nil {
		return errors.New("effective config is nil")
	}
	if match == nil {
		return errors.New("overwrite matcher is nil")
	}
	if apply == nil {
		return errors.New("overwrite applier is nil")
	}

	for i := range overwrites {
		ow := &overwrites[i]
		matched, err := match(ow, target)
		if err != nil {
			return fmt.Errorf("overwrites[%d]: %w", i, err)
		}
		if matched {
			apply(effective, ow)
		}
	}

	return nil
}

// Resolve returns a copy of base with matching overwrites applied.
func Resolve[E, O, T any](
	base E,
	target T,
	overwrites []O,
	match Matcher[O, T],
	apply Applier[E, O],
) (E, error) {
	effective := base
	if err := ApplyMatching(&effective, target, overwrites, match, apply); err != nil {
		return effective, err
	}
	return effective, nil
}

// ByKey matches an overwrite by an extracted comparable key.
func ByKey[O any, K comparable](key func(*O) K) Matcher[O, K] {
	return func(ow *O, target K) (bool, error) {
		if key == nil {
			return false, errors.New("overwrite key extractor is nil")
		}
		return key(ow) == target, nil
	}
}

// ByPathGlob matches a slash-separated path using a glob pattern extracted
// from the overwrite. The glob supports ** as a whole path segment.
func ByPathGlob[O any](pattern func(*O) string) Matcher[O, string] {
	return func(ow *O, target string) (bool, error) {
		if pattern == nil {
			return false, errors.New("overwrite pattern extractor is nil")
		}
		return MatchPathGlob(pattern(ow), target)
	}
}

// ValidatePathGlob validates a slash-separated glob pattern. ** is valid only
// as a whole segment and matches zero or more path segments.
func ValidatePathGlob(pattern string) error {
	return pathmatch.ValidateGlob(pattern)
}

// MatchPathGlob reports whether path matches pattern. It returns malformed
// glob errors instead of silently treating them as non-matches.
func MatchPathGlob(pattern, path string) (bool, error) {
	return pathmatch.MatchGlob(pattern, path)
}
