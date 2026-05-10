package policy

import "github.com/edelwud/terraci/plugins/policy/internal/engine"

// VersionInfo contributes OPA version to `terraci version`.
func (p *Plugin) VersionInfo() map[string]string {
	return map[string]string{"opa": engine.OPAVersion()}
}
