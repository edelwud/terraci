package tfupdateengine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Version represents a semantic version.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

// String returns the version as "major.minor.patch" (or with prerelease).
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	return s
}

// IsZero returns true if the version is 0.0.0.
func (v Version) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0 && v.Prerelease == ""
}

// Compare returns -1, 0, or 1.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return cmpInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return cmpInt(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return cmpInt(v.Patch, other.Patch)
	}
	// A version with prerelease has lower precedence than without.
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	return strings.Compare(v.Prerelease, other.Prerelease)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	return 1
}

// ParseVersion parses a semver string like "1.2.3" or "1.2.3-beta".
func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimSpace(s)

	var v Version
	var prerelease string

	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		prerelease = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return v, fmt.Errorf("invalid version %q", s)
	}

	var err error
	v.Major, err = strconv.Atoi(parts[0])
	if err != nil {
		return v, fmt.Errorf("invalid major version: %w", err)
	}
	if len(parts) >= 2 {
		v.Minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return v, fmt.Errorf("invalid minor version: %w", err)
		}
	}
	if len(parts) >= 3 {
		v.Patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return v, fmt.Errorf("invalid patch version: %w", err)
		}
	}
	v.Prerelease = prerelease
	return v, nil
}

// ConstraintOp represents a version constraint operator.
type ConstraintOp int

const (
	OpEqual        ConstraintOp = iota // =
	OpNotEqual                         // !=
	OpGreater                          // >
	OpGreaterEqual                     // >=
	OpLess                             // <
	OpLessEqual                        // <=
	OpPessimistic                      // ~>
)

// Constraint represents a single version constraint like "~> 5.0".
type Constraint struct {
	Op      ConstraintOp
	Version Version
	// Raw parts count to distinguish "~> 5.0" (2 parts) from "~> 5.0.1" (3 parts).
	Parts int
}

// ParseConstraints parses comma-separated version constraints.
// Examples: "~> 5.0", ">= 1.0, < 2.0", "= 1.2.3"
func ParseConstraints(s string) ([]Constraint, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	constraints := make([]Constraint, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		c, err := parseSingleConstraint(part)
		if err != nil {
			return nil, fmt.Errorf("parse constraint %q: %w", part, err)
		}
		constraints = append(constraints, c)
	}
	return constraints, nil
}

func parseSingleConstraint(s string) (Constraint, error) {
	s = strings.TrimSpace(s)
	var c Constraint

	switch {
	case strings.HasPrefix(s, "~>"):
		c.Op = OpPessimistic
		s = strings.TrimSpace(s[2:])
	case strings.HasPrefix(s, ">="):
		c.Op = OpGreaterEqual
		s = strings.TrimSpace(s[2:])
	case strings.HasPrefix(s, "<="):
		c.Op = OpLessEqual
		s = strings.TrimSpace(s[2:])
	case strings.HasPrefix(s, "!="):
		c.Op = OpNotEqual
		s = strings.TrimSpace(s[2:])
	case strings.HasPrefix(s, ">"):
		c.Op = OpGreater
		s = strings.TrimSpace(s[1:])
	case strings.HasPrefix(s, "<"):
		c.Op = OpLess
		s = strings.TrimSpace(s[1:])
	case strings.HasPrefix(s, "="):
		c.Op = OpEqual
		s = strings.TrimSpace(s[1:])
	default:
		// No operator means exact match.
		c.Op = OpEqual
	}

	c.Parts = len(strings.Split(strings.TrimPrefix(s, "v"), "."))

	v, err := ParseVersion(s)
	if err != nil {
		return c, err
	}
	c.Version = v
	return c, nil
}

// Satisfies checks if a version satisfies a single constraint.
func (c Constraint) Satisfies(v Version) bool {
	switch c.Op {
	case OpEqual:
		return v.Compare(c.Version) == 0
	case OpNotEqual:
		return v.Compare(c.Version) != 0
	case OpGreater:
		return v.Compare(c.Version) > 0
	case OpGreaterEqual:
		return v.Compare(c.Version) >= 0
	case OpLess:
		return v.Compare(c.Version) < 0
	case OpLessEqual:
		return v.Compare(c.Version) <= 0
	case OpPessimistic:
		return c.satisfiesPessimistic(v)
	}
	return false
}

func (c Constraint) satisfiesPessimistic(v Version) bool {
	// ~> 5.0 means >= 5.0, < 6.0 (bumps major)
	// ~> 5.0.1 means >= 5.0.1, < 5.1.0 (bumps minor)
	if v.Compare(c.Version) < 0 {
		return false
	}

	if c.Parts <= 2 {
		// ~> X.Y: upper bound is (X+1).0.0
		return v.Major == c.Version.Major
	}
	// ~> X.Y.Z: upper bound is X.(Y+1).0
	return v.Major == c.Version.Major && v.Minor == c.Version.Minor
}

// SatisfiesAll checks if a version satisfies all constraints.
func SatisfiesAll(v Version, constraints []Constraint) bool {
	for _, c := range constraints {
		if !c.Satisfies(v) {
			return false
		}
	}
	return true
}

// LatestAllowed finds the highest version from the list that satisfies all constraints.
func LatestAllowed(versions []Version, constraints []Constraint) (Version, bool) {
	// Sort descending.
	sorted := make([]Version, len(versions))
	copy(sorted, versions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Compare(sorted[j]) > 0
	})

	for _, v := range sorted {
		if v.Prerelease != "" {
			continue // Skip prereleases.
		}
		if SatisfiesAll(v, constraints) {
			return v, true
		}
	}
	return Version{}, false
}

// LatestByBump finds the highest version respecting the bump level policy.
// - patch: only allow same major.minor
// - minor: only allow same major
// - major: allow anything (latest absolute)
func LatestByBump(current Version, versions []Version, bump string) (Version, bool) {
	sorted := make([]Version, len(versions))
	copy(sorted, versions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Compare(sorted[j]) > 0
	})

	for _, v := range sorted {
		if v.Prerelease != "" {
			continue
		}
		if v.Compare(current) <= 0 {
			continue
		}
		switch bump {
		case BumpPatch:
			if v.Major == current.Major && v.Minor == current.Minor {
				return v, true
			}
		case BumpMinor:
			if v.Major == current.Major {
				return v, true
			}
		case BumpMajor:
			return v, true
		}
	}
	return Version{}, false
}

// BumpConstraint generates a new constraint string based on the original style and new version.
func BumpConstraint(original string, newVersion Version) string {
	original = strings.TrimSpace(original)

	constraints, err := ParseConstraints(original)
	if err != nil || len(constraints) == 0 {
		return original
	}

	// For simple single constraints, preserve operator style.
	if len(constraints) == 1 {
		c := constraints[0]
		switch c.Op { //nolint:exhaustive // only rewrite known operator styles
		case OpPessimistic:
			if c.Parts <= 2 {
				return fmt.Sprintf("~> %d.%d", newVersion.Major, newVersion.Minor)
			}
			return fmt.Sprintf("~> %d.%d.%d", newVersion.Major, newVersion.Minor, newVersion.Patch)
		case OpEqual:
			if c.Parts <= 2 {
				return fmt.Sprintf("= %d.%d", newVersion.Major, newVersion.Minor)
			}
			return fmt.Sprintf("= %d.%d.%d", newVersion.Major, newVersion.Minor, newVersion.Patch)
		case OpGreaterEqual:
			if c.Parts <= 2 {
				return fmt.Sprintf(">= %d.%d", newVersion.Major, newVersion.Minor)
			}
			return fmt.Sprintf(">= %d.%d.%d", newVersion.Major, newVersion.Minor, newVersion.Patch)
		}
	}

	// For complex constraints, return pessimistic based on new version.
	return fmt.Sprintf("~> %d.%d", newVersion.Major, newVersion.Minor)
}
