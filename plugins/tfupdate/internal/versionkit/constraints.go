package versionkit

import (
	"fmt"
	"sort"
	"strings"
)

// MergeConstraints combines a root constraint with additional constraints from
// child modules, producing a comma-separated deduplicated normalized string
// matching Terraform's lock file format.
func MergeConstraints(root string, extras []string) string {
	all := make([]string, 0, 1+len(extras))
	seen := make(map[string]struct{})

	for _, c := range append(extras, root) {
		for _, normalized := range normalizeConstraintParts(c) {
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			all = append(all, normalized)
		}
	}

	if len(all) == 0 {
		return root
	}

	sort.Slice(all, constraintLess(all))
	return strings.Join(all, ", ")
}

var constraintOpStr = map[ConstraintOp]string{
	OpEqual:        "",
	OpNotEqual:     "!= ",
	OpGreater:      "> ",
	OpGreaterEqual: ">= ",
	OpLess:         "< ",
	OpLessEqual:    "<= ",
	OpPessimistic:  "~> ",
}

func constraintLess(items []string) func(i, j int) bool {
	type parsed struct {
		v     Version
		hasOp bool
	}
	cache := make(map[string]parsed, len(items))
	for _, s := range items {
		c, err := parseSingleConstraint(s)
		if err != nil {
			continue
		}
		cache[s] = parsed{v: c.Version, hasOp: c.Op != OpEqual}
	}

	return func(i, j int) bool {
		ci, cj := cache[items[i]], cache[items[j]]
		if cmp := ci.v.Compare(cj.v); cmp != 0 {
			return cmp < 0
		}
		if ci.hasOp != cj.hasOp {
			return ci.hasOp
		}
		return items[i] < items[j]
	}
}

func normalizeConstraintParts(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	constraints, err := ParseConstraints(s)
	if err != nil {
		return []string{s}
	}

	out := make([]string, 0, len(constraints))
	for _, c := range constraints {
		prefix := constraintOpStr[c.Op]
		ver := c.Version.String()

		if c.Op == OpPessimistic && c.Parts <= 2 {
			ver = fmt.Sprintf("%d.%d", c.Version.Major, c.Version.Minor)
		}

		out = append(out, prefix+ver)
	}
	return out
}
