package update

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes dependency update fields to the init wizard.

const initGroupOrder = 202

// InitGroups returns the init wizard group specs for the update plugin.
func (p *Plugin) InitGroups() []*plugin.InitGroupSpec {
	return []*plugin.InitGroupSpec{
		{
			Title:    "Dependency Updates",
			Category: plugin.CategoryFeature,
			Order:    initGroupOrder,
			Fields: []plugin.InitField{
				{
					Key:         "update.enabled",
					Title:       "Enable dependency update checks?",
					Description: "Check for outdated Terraform provider and module versions",
					Type:        "bool",
					Default:     false,
				},
			},
		},
		{
			Title:    "Update Settings",
			Category: plugin.CategoryDetail,
			Order:    initGroupOrder,
			ShowWhen: func(s *plugin.StateMap) bool {
				return s.Bool("update.enabled")
			},
			Fields: []plugin.InitField{
				{
					Key:     "update.target",
					Title:   "What to check",
					Type:    "select",
					Default: "all",
					Options: []plugin.InitOption{
						{Label: "All (modules + providers)", Value: "all"},
						{Label: "Modules only", Value: "modules"},
						{Label: "Providers only", Value: "providers"},
					},
				},
				{
					Key:     "update.bump",
					Title:   "Maximum bump level",
					Type:    "select",
					Default: "minor",
					Options: []plugin.InitOption{
						{Label: "Patch only", Value: "patch"},
						{Label: "Minor", Value: "minor"},
						{Label: "Major", Value: "major"},
					},
				},
				{
					Key:         "update.pipeline",
					Title:       "Add update check to CI pipeline?",
					Description: "Add a dependency-update-check job to generated pipelines",
					Type:        "bool",
					Default:     false,
				},
			},
		},
	}
}

// BuildInitConfig builds the update init contribution.
func (p *Plugin) BuildInitConfig(state *plugin.StateMap) *plugin.InitContribution {
	enabled := state.Bool("update.enabled")
	if !enabled {
		return nil
	}

	cfg := map[string]any{
		"enabled": true,
	}

	if target := state.String("update.target"); target != "" && target != "all" {
		cfg["target"] = target
	}
	if bump := state.String("update.bump"); bump != "" && bump != "minor" {
		cfg["bump"] = bump
	}
	if state.Bool("update.pipeline") {
		cfg["pipeline"] = true
	}

	return &plugin.InitContribution{
		PluginKey: "update",
		Config:    cfg,
	}
}
