package policy

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes policy check fields to the init wizard.

const initGroupOrder = 201

// InitGroups returns the init wizard group specs for policy checks.
// Two groups: a feature toggle and a detail group for settings.
func (p *Plugin) InitGroups() []*plugin.InitGroupSpec {
	return []*plugin.InitGroupSpec{
		{
			Title:    "Policy Checks",
			Category: plugin.CategoryFeature,
			Order:    initGroupOrder,
			Fields: []plugin.InitField{
				{
					Key:         "policy.enabled",
					Title:       "Enable policy checks?",
					Description: "Evaluate Terraform plans with OPA policies",
					Type:        "bool",
					Default:     false,
				},
			},
		},
		{
			Title:    "Policy Settings",
			Category: plugin.CategoryDetail,
			Order:    initGroupOrder,
			ShowWhen: func(s *plugin.StateMap) bool {
				return s.Bool("policy.enabled")
			},
			Fields: []plugin.InitField{
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
		},
	}
}

// BuildInitConfig builds the policy checks init contribution.
func (p *Plugin) BuildInitConfig(state *plugin.StateMap) *plugin.InitContribution {
	enabled := state.Bool("policy.enabled")
	if !enabled {
		return nil
	}

	sourcePath := state.String("policy.source_path")
	if sourcePath == "" {
		sourcePath = "policies"
	}

	onFailure := state.String("policy.on_failure")
	if onFailure == "" {
		onFailure = "block"
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
