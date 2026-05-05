// Package summary provides the summary plugin for TerraCi.
package summary

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// InitContributor — contributes summary field to the init wizard.

const initGroupOrder = 199 // before cost/policy so it appears first in Features

// InitGroups returns the summary plugin's form group for the init wizard.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	return []*initwiz.InitGroupSpec{
		{
			Title:    "Summary",
			Category: initwiz.CategoryFeature,
			Order:    initGroupOrder,
			Fields: []initwiz.InitField{
				{
					Key:         "summary.enabled",
					Title:       "Enable plan summaries?",
					Description: "Post Terraform plan summaries as merge and pull request comments",
					Type:        initwiz.FieldBool,
					Default:     true,
				},
			},
		},
	}
}

// BuildInitConfig builds the summary plugin config from wizard state.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) *initwiz.InitContribution {
	enabled := state.Bool("summary.enabled")
	return &initwiz.InitContribution{
		PluginKey: pluginName,
		Config: map[string]any{
			"enabled": enabled,
		},
	}
}
