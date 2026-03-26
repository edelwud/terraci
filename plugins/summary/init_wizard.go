package summary

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes summary field to the init wizard.

const initGroupOrder = 200

// InitGroup returns the summary plugin's form group for the init wizard.
func (p *Plugin) InitGroup() *plugin.InitGroupSpec {
	return &plugin.InitGroupSpec{
		Title: "Summary",
		Order: initGroupOrder,
		Fields: []plugin.InitField{
			{
				Key:         "summary.enabled",
				Title:       "Enable MR/PR comments?",
				Description: "Post plan summaries as comments on merge/pull requests",
				Type:        "bool",
				Default:     true,
			},
		},
	}
}

// BuildInitConfig builds the summary plugin config from wizard state.
func (p *Plugin) BuildInitConfig(state plugin.InitState) *plugin.InitContribution {
	enabled, ok := state.Get("summary.enabled").(bool)
	if !ok || !enabled {
		return nil
	}
	return &plugin.InitContribution{
		PluginKey: "summary",
		Config:    map[string]any{},
	}
}
