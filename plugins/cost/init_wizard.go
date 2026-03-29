package cost

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes cost estimation field to the init wizard.

const initGroupOrder = 200

// InitGroups returns the init wizard group spec for cost estimation.
func (p *Plugin) InitGroups() []*plugin.InitGroupSpec {
	return []*plugin.InitGroupSpec{
		{
			Title:    "Cost Estimation",
			Category: plugin.CategoryFeature,
			Order:    initGroupOrder,
			Fields: []plugin.InitField{
				{
					Key:         "cost.enabled",
					Title:       "Enable cost estimation?",
					Description: "Estimate AWS costs from Terraform plans",
					Type:        "bool",
					Default:     false,
				},
			},
		},
	}
}

// BuildInitConfig builds the cost estimation init contribution.
func (p *Plugin) BuildInitConfig(state *plugin.StateMap) *plugin.InitContribution {
	enabled := state.Bool("cost.enabled")
	if !enabled {
		return nil
	}
	return &plugin.InitContribution{
		PluginKey: "cost",
		Config: map[string]any{
			"enabled": true,
		},
	}
}
