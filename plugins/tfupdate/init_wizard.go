package tfupdate

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// InitContributor — contributes dependency update fields to the init wizard.

const initGroupOrder = 202

// Wizard StateMap keys. Centralized so InitGroups field definitions and
// BuildInitConfig consumers can never drift apart on a typo. Note that the
// keys keep the historical "update." prefix for state compatibility — the
// plugin name is "tfupdate" but its wizard fields shipped under "update.*".
const (
	keyUpdateEnabled  = "update.enabled"
	keyUpdateTarget   = "update.target"
	keyUpdateBump     = "update.bump"
	keyUpdatePipeline = "update.pipeline"
)

// InitGroups returns the init wizard group specs for the update initwiz.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	return []*initwiz.InitGroupSpec{
		{
			Title:    "Dependency Updates",
			Category: initwiz.CategoryFeature,
			Order:    initGroupOrder,
			Fields: []initwiz.InitField{
				{
					Key:         keyUpdateEnabled,
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
				return s.Bool(keyUpdateEnabled)
			},
			Fields: []initwiz.InitField{
				{
					Key:     keyUpdateTarget,
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
					Key:     keyUpdateBump,
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
					Key:         keyUpdatePipeline,
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
	enabled := state.Bool(keyUpdateEnabled)
	if !enabled {
		return nil
	}

	cfg := map[string]any{
		"enabled": true,
	}

	if target := state.String(keyUpdateTarget); target != "" && target != "all" {
		cfg["target"] = target
	}
	if bump := state.String(keyUpdateBump); bump != "" && bump != "minor" {
		cfg["bump"] = bump
	}
	if state.Bool(keyUpdatePipeline) {
		cfg["pipeline"] = true
	}

	return &initwiz.InitContribution{
		PluginKey: pluginName,
		Config:    cfg,
	}
}
