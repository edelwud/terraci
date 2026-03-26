package policy

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes policy check fields to the init wizard.

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
			{
				Key:         "policy.source_path",
				Title:       "Policy files directory",
				Description: "Local directory containing .rego policy files",
				Type:        "string",
				Default:     "policies",
				Placeholder: "policies",
			},
			{
				Key:         "policy.on_failure",
				Title:       "On policy failure",
				Description: "Action when policy check fails",
				Type:        "select",
				Default:     "block",
				Options: []plugin.InitOption{
					{Label: "Block pipeline", Value: "block"},
					{Label: "Warn only", Value: "warn"},
				},
			},
		},
	}
}

// BuildInitConfig builds the policy checks init contribution.
func (p *Plugin) BuildInitConfig(state plugin.InitState) *plugin.InitContribution {
	enabled, ok := state.Get("policy.enabled").(bool)
	if !ok || !enabled {
		return nil
	}

	sourcePath := "policies"
	if v, ok := state.Get("policy.source_path").(string); ok && v != "" {
		sourcePath = v
	}

	onFailure := "block"
	if v, ok := state.Get("policy.on_failure").(string); ok && v != "" {
		onFailure = v
	}

	return &plugin.InitContribution{
		PluginKey: "policy",
		Config: map[string]any{
			"enabled":    true,
			"sources":    []map[string]any{{"path": sourcePath}},
			"on_failure": onFailure,
		},
	}
}
