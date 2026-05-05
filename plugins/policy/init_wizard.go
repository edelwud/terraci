package policy

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// InitContributor — contributes policy check fields to the init wizard.

const initGroupOrder = 201

// Wizard StateMap keys. Centralized so InitGroups field definitions and
// BuildInitConfig consumers can never drift apart on a typo.
const (
	keyPolicyEnabled    = "policy.enabled"
	keyPolicySourcePath = "policy.source_path"
	keyPolicyOnFailure  = "policy.on_failure"
)

// InitGroups returns the init wizard group specs for policy checks.
// Two groups: a feature toggle and a detail group for settings.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	return []*initwiz.InitGroupSpec{
		{
			Title:    "Policy Checks",
			Category: initwiz.CategoryFeature,
			Order:    initGroupOrder,
			Fields: []initwiz.InitField{
				{
					Key:         keyPolicyEnabled,
					Title:       "Enable policy checks?",
					Description: "Evaluate Terraform plans with OPA policies",
					Type:        initwiz.FieldBool,
					Default:     false,
				},
			},
		},
		{
			Title:    "Policy Settings",
			Category: initwiz.CategoryDetail,
			Order:    initGroupOrder,
			ShowWhen: func(s *initwiz.StateMap) bool {
				return s.Bool(keyPolicyEnabled)
			},
			Fields: []initwiz.InitField{
				{
					Key:         keyPolicySourcePath,
					Title:       "Policy files directory",
					Description: "Local directory containing .rego policy files",
					Type:        initwiz.FieldString,
					Default:     "policies",
					Placeholder: "policies",
				},
				{
					Key:         keyPolicyOnFailure,
					Title:       "On policy failure",
					Description: "Action when policy check fails",
					Type:        initwiz.FieldSelect,
					Default:     "block",
					Options: []initwiz.InitOption{
						{Label: "Block pipeline", Value: "block"},
						{Label: "Warn only", Value: "warn"},
					},
				},
			},
		},
	}
}

// BuildInitConfig builds the policy checks init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) *initwiz.InitContribution {
	enabled := state.Bool(keyPolicyEnabled)
	if !enabled {
		return nil
	}

	sourcePath := state.String(keyPolicySourcePath)
	if sourcePath == "" {
		sourcePath = "policies"
	}

	onFailure := state.String(keyPolicyOnFailure)
	if onFailure == "" {
		onFailure = "block"
	}

	return &initwiz.InitContribution{
		PluginKey: "policy",
		Config: map[string]any{
			"enabled":    true,
			"sources":    []map[string]any{{"path": sourcePath}},
			"on_failure": onFailure,
		},
	}
}
