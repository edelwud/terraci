package policy

import (
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

// InitContributor — contributes policy check fields to the init wizard.

const initGroupOrder = 201

var (
	keyPolicyEnabled      = initwiz.MustStateKey[bool]("policy.enabled")
	keyPolicySourcePath   = initwiz.MustStateKey[string]("policy.source_path")
	keyPolicyDenyDecision = initwiz.MustStateKey[string]("policy.decisions.deny")
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
				initwiz.NewBoolField(initwiz.BoolFieldOptions{
					Key:         keyPolicyEnabled,
					Title:       "Enable policy checks?",
					Description: "Evaluate Terraform plans with OPA policies",
					Default:     false,
				}),
			},
		},
		{
			Title:    "Policy Settings",
			Category: initwiz.CategoryDetail,
			Order:    initGroupOrder,
			ShowWhen: keyPolicyEnabled.Get,
			Fields: []initwiz.InitField{
				initwiz.NewStringField(initwiz.StringFieldOptions{
					Key:         keyPolicySourcePath,
					Title:       "Policy files directory",
					Description: "Local directory containing .rego policy files",
					Default:     "policies",
					Placeholder: "policies",
				}),
				initwiz.NewSelectField(initwiz.SelectFieldOptions{
					Key:         keyPolicyDenyDecision,
					Title:       "On deny decisions",
					Description: "Action when OPA deny rules match",
					Default:     "block",
					Options: []initwiz.InitOption{
						{Label: "Block pipeline", Value: "block"},
						{Label: "Warn only", Value: "warn"},
						{Label: "Ignore", Value: "ignore"},
					},
				}),
			},
		},
	}
}

// BuildInitConfig builds the policy checks init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
	enabled := keyPolicyEnabled.Get(state)
	if !enabled {
		return nil, nil
	}

	sourcePath := keyPolicySourcePath.Get(state)
	if sourcePath == "" {
		sourcePath = "policies"
	}

	denyAction := keyPolicyDenyDecision.Get(state)
	if denyAction == "" {
		denyAction = "block"
	}

	return initwiz.NewInitContribution(pluginName, &policyengine.Config{
		Enabled: true,
		Sources: []policyengine.SourceConfig{
			{Type: policyengine.SourceTypePath, Path: sourcePath},
		},
		Decisions: policyengine.Decisions{Deny: policyengine.Action(denyAction)},
	})
}
