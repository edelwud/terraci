package policy

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// InitContributor — contributes policy check fields to the init wizard.

const initGroupOrder = 201

// Wizard StateMap keys. Centralized so InitGroups field definitions and
// BuildInitConfig consumers can never drift apart on a typo.
const (
	keyPolicyEnabled      = "policy.enabled"
	keyPolicySourcePath   = "policy.source_path"
	keyPolicyDenyDecision = "policy.decisions.deny"
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
					Key:         keyPolicyDenyDecision,
					Title:       "On deny decisions",
					Description: "Action when OPA deny rules match",
					Type:        initwiz.FieldSelect,
					Default:     "block",
					Options: []initwiz.InitOption{
						{Label: "Block pipeline", Value: "block"},
						{Label: "Warn only", Value: "warn"},
						{Label: "Ignore", Value: "ignore"},
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

	denyAction := state.String(keyPolicyDenyDecision)
	if denyAction == "" {
		denyAction = "block"
	}

	return &initwiz.InitContribution{
		PluginKey: pluginName,
		Config: map[string]any{
			"enabled":   true,
			"sources":   []map[string]any{{"type": "path", "path": sourcePath}},
			"decisions": map[string]any{"deny": denyAction},
		},
	}
}
