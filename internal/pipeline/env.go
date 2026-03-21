package pipeline

import (
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
)

// BuildModuleEnvVars creates environment variables for a module dynamically from its segments.
func BuildModuleEnvVars(module *discovery.Module) map[string]string {
	env := map[string]string{
		"TF_MODULE_PATH": module.RelativePath,
		"TF_MODULE":      module.Name(),
	}
	for _, seg := range module.Segments() {
		env["TF_"+strings.ToUpper(seg)] = module.Get(seg)
	}
	return env
}
