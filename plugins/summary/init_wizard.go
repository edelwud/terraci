// Package summary provides the summary plugin for TerraCi.
package summary

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes summary field to the init wizard.

const initGroupOrder = 199 // before cost/policy so it appears first in Features

// InitGroups returns the summary plugin's form group for the init wizard.
func (p *Plugin) InitGroups() []*plugin.InitGroupSpec {
	return []*plugin.InitGroupSpec{
		{
			Title:    "Summary",
			Category: plugin.CategoryFeature,
			Order:    initGroupOrder,
			Fields: []plugin.InitField{
				{
					Key:         "summary.enabled",
					Title:       "Enable plan summary & MR/PR comments?",
					Description: "Post plan summaries as comments on merge/pull requests",
					Type:        "bool",
					Default:     true,
				},
			},
		},
	}
}

// BuildInitConfig builds the summary plugin config from wizard state.
func (p *Plugin) BuildInitConfig(state *plugin.StateMap) *plugin.InitContribution {
	enabled := state.Bool("summary.enabled")
	if !enabled {
		return nil
	}
	return &plugin.InitContribution{
		PluginKey: "summary",
		Config:    map[string]any{},
	}
}
