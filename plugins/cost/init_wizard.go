package cost

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes cost estimation field to the init wizard.

const initGroupOrder = 200

// InitGroup returns the init wizard group spec for cost estimation.
func (p *Plugin) InitGroup() *plugin.InitGroupSpec {
	return &plugin.InitGroupSpec{
		Title: "Cost Estimation",
		Order: initGroupOrder,
		Fields: []plugin.InitField{
			{
				Key:         "cost.enabled",
				Title:       "Enable cost estimation?",
				Description: "Estimate AWS costs from Terraform plans",
				Type:        "bool",
				Default:     false,
			},
		},
	}
}

// BuildInitConfig builds the cost estimation init contribution.
func (p *Plugin) BuildInitConfig(state plugin.InitState) *plugin.InitContribution {
	enabled, ok := state.Get("cost.enabled").(bool)
	if !ok {
		return nil
	}
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
