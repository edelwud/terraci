package pipeline

import (
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
)

// ModuleEnvVars derives standard TF_* environment variables from a module's
// path segments. Used by both pipeline generation and local execution to keep
// the variable surface consistent.
func ModuleEnvVars(module *discovery.Module) map[string]string {
	env := map[string]string{
		"TF_MODULE_PATH": module.RelativePath,
		"TF_MODULE":      module.Name(),
	}
	for _, seg := range module.Segments() {
		env["TF_"+strings.ToUpper(seg)] = module.Get(seg)
	}
	return env
}
