package cmd

import (
	"fmt"
	"sort"
	"strings"
)

// BuiltinPlugins lists the import paths of all built-in plugins.
// These are included by default unless excluded with --without.
var BuiltinPlugins = map[string]string{
	"gitlab": "github.com/edelwud/terraci/plugins/gitlab",
	"github": "github.com/edelwud/terraci/plugins/github",
	"cost":   "github.com/edelwud/terraci/plugins/cost",
	"policy": "github.com/edelwud/terraci/plugins/policy",
	"git":    "github.com/edelwud/terraci/plugins/git",
}

// builtinNames returns a sorted, comma-separated list of built-in plugin names.
func builtinNames() string {
	names := make([]string, 0, len(BuiltinPlugins))
	for k := range BuiltinPlugins {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// validateWithout checks that every name in names exists in BuiltinPlugins.
func validateWithout(names []string) error {
	for _, name := range names {
		if _, ok := BuiltinPlugins[name]; !ok {
			return fmt.Errorf("unknown plugin %q (available: %s)", name, builtinNames())
		}
	}
	return nil
}

// validateWith checks that every --with spec has a valid module path (must contain "/").
func validateWith(specs []string) error {
	for _, s := range specs {
		mod := s
		// Strip replacement
		if idx := strings.Index(mod, "="); idx >= 0 {
			mod = mod[:idx]
		}
		// Strip version
		if idx := strings.Index(mod, "@"); idx >= 0 {
			mod = mod[:idx]
		}
		if !strings.Contains(mod, "/") {
			return fmt.Errorf("invalid module path %q: must contain \"/\" (e.g. github.com/org/plugin)", mod)
		}
	}
	return nil
}
