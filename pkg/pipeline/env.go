package pipeline

import (
	"maps"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
)

// ModuleEnvVars derives standard TF_* environment variables from a module's
// path segments. Used by both pipeline generation and local execution to keep
// the variable surface consistent.
func ModuleEnvVars(module *discovery.Module) map[string]string {
	env := map[string]string{
		"TF_MODULE_PATH": module.ID(),
		"TF_MODULE":      module.Name(),
	}
	for _, seg := range module.Segments() {
		env["TF_"+strings.ToUpper(seg)] = module.Get(seg)
	}
	return env
}

// TerraformJobEnv merges execution-level environment with canonical module
// TF_* variables. Module-derived values win on key conflicts.
func TerraformJobEnv(executionEnv map[string]string, module *discovery.Module) map[string]string {
	env := make(map[string]string, len(executionEnv)+8)
	maps.Copy(env, executionEnv)
	maps.Copy(env, ModuleEnvVars(module))
	return env
}
