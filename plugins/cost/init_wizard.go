package cost

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// InitContributor — contributes cost estimation field to the init wizard.

const initGroupOrder = 200

// InitGroups returns the init wizard group spec for cost estimation.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	return []*initwiz.InitGroupSpec{
		{
			Title:    "Cost Estimation",
			Category: initwiz.CategoryFeature,
			Order:    initGroupOrder,
			Fields: []initwiz.InitField{
				{
					Key:         "cost.providers.aws.enabled",
					Title:       "Enable cloud cost estimation?",
					Description: "Estimate cloud costs from Terraform plans",
					Type:        initwiz.FieldBool,
					Default:     false,
				},
			},
		},
	}
}

// BuildInitConfig builds the cost estimation init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) *initwiz.InitContribution {
	enabled := state.Bool("cost.providers.aws.enabled")
	if !enabled {
		return nil
	}
	return &initwiz.InitContribution{
		PluginKey: "cost",
		Config: map[string]any{
			"providers": map[string]any{
				"aws": map[string]any{
					"enabled": true,
				},
			},
		},
	}
}
