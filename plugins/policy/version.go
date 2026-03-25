package policy

import policyengine "github.com/edelwud/terraci/plugins/policy/internal"

// VersionInfo contributes OPA version to `terraci version`.
func (p *Plugin) VersionInfo() map[string]string {
	return map[string]string{"opa": policyengine.OPAVersion()}
}
