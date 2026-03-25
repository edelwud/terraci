package policy

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes policy check field to the init wizard.

const initGroupOrder = 201

// InitGroup returns the init wizard group spec for policy checks.
func (p *Plugin) InitGroup() *plugin.InitGroupSpec {
	return &plugin.InitGroupSpec{
		Title: "Policy Checks",
		Order: initGroupOrder,
		Fields: []plugin.InitField{
			{
				Key:         "policy.enabled",
				Title:       "Enable policy checks?",
				Description: "Run OPA policy checks against Terraform plans",
				Type:        "bool",
				Default:     false,
			},
		},
	}
}

// BuildInitConfig builds the policy checks init contribution.
func (p *Plugin) BuildInitConfig(state plugin.InitState) *plugin.InitContribution {
	enabled, ok := state.Get("policy.enabled").(bool)
	if !ok {
		return nil
	}
	if !enabled {
		return nil
	}
	return &plugin.InitContribution{
		PluginKey: "policy",
		Config: map[string]any{
			"enabled": true,
		},
	}
}
