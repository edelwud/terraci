package tfupdate

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// InitContributor — contributes dependency update fields to the init wizard.

const initGroupOrder = 202

// InitGroups returns the init wizard group specs for the update initwiz.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	return []*initwiz.InitGroupSpec{
		{
			Title:    "Dependency Updates",
			Category: initwiz.CategoryFeature,
			Order:    initGroupOrder,
			Fields: []initwiz.InitField{
				{
					Key:         "update.enabled",
					Title:       "Enable dependency update checks?",
					Description: "Check Terraform providers and modules for newer versions",
					Type:        initwiz.FieldBool,
					Default:     false,
				},
			},
		},
		{
			Title:    "Update Settings",
			Category: initwiz.CategoryDetail,
			Order:    initGroupOrder,
			ShowWhen: func(s *initwiz.StateMap) bool {
				return s.Bool("update.enabled")
			},
			Fields: []initwiz.InitField{
				{
					Key:     "update.target",
					Title:   "What to check",
					Type:    initwiz.FieldSelect,
					Default: "all",
					Options: []initwiz.InitOption{
						{Label: "All (modules + providers)", Value: "all"},
						{Label: "Modules only", Value: "modules"},
						{Label: "Providers only", Value: "providers"},
					},
				},
				{
					Key:     "update.bump",
					Title:   "Maximum bump level",
					Type:    initwiz.FieldSelect,
					Default: "minor",
					Options: []initwiz.InitOption{
						{Label: "Patch only", Value: "patch"},
						{Label: "Minor", Value: "minor"},
						{Label: "Major", Value: "major"},
					},
				},
				{
					Key:         "update.pipeline",
					Title:       "Add update check to CI pipeline?",
					Description: "Add a tfupdate-check job to generated pipelines",
					Type:        initwiz.FieldBool,
					Default:     false,
				},
			},
		},
	}
}

// BuildInitConfig builds the update init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) *initwiz.InitContribution {
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

	return &initwiz.InitContribution{
		PluginKey: "tfupdate",
		Config:    cfg,
	}
}
