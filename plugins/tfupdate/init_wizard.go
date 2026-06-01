package tfupdate

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// InitContributor — contributes dependency update fields to the init wizard.

const initGroupOrder = 202

var (
	keyUpdateEnabled  = initwiz.MustStateKey[bool]("tfupdate.enabled")
	keyUpdateTarget   = initwiz.MustStateKey[string]("tfupdate.target")
	keyUpdateBump     = initwiz.MustStateKey[string]("tfupdate.bump")
	keyUpdatePipeline = initwiz.MustStateKey[bool]("tfupdate.pipeline")
)

type initConfig struct {
	Enabled  bool              `yaml:"enabled"`
	Target   string            `yaml:"target,omitempty"`
	Policy   *initPolicyConfig `yaml:"policy,omitempty"`
	Pipeline bool              `yaml:"pipeline,omitempty"`
}

type initPolicyConfig struct {
	Bump string `yaml:"bump,omitempty"`
}

// InitGroups returns the init wizard group specs for the update initwiz.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	return []*initwiz.InitGroupSpec{
		{
			Title:    "Dependency Updates",
			Category: initwiz.CategoryFeature,
			Order:    initGroupOrder,
			Fields: []initwiz.InitField{
				initwiz.NewBoolField(initwiz.BoolFieldOptions{
					Key:         keyUpdateEnabled,
					Title:       "Enable dependency update checks?",
					Description: "Check Terraform providers and modules for newer versions",
					Default:     false,
				}),
			},
		},
		{
			Title:    "Update Settings",
			Category: initwiz.CategoryDetail,
			Order:    initGroupOrder,
			ShowWhen: keyUpdateEnabled.Get,
			Fields: []initwiz.InitField{
				initwiz.NewSelectField(initwiz.SelectFieldOptions{
					Key:     keyUpdateTarget,
					Title:   "What to check",
					Default: "all",
					Options: []initwiz.InitOption{
						{Label: "All (modules + providers)", Value: "all"},
						{Label: "Modules only", Value: "modules"},
						{Label: "Providers only", Value: "providers"},
					},
				}),
				initwiz.NewSelectField(initwiz.SelectFieldOptions{
					Key:     keyUpdateBump,
					Title:   "Maximum bump level",
					Default: "minor",
					Options: []initwiz.InitOption{
						{Label: "Patch only", Value: "patch"},
						{Label: "Minor", Value: "minor"},
						{Label: "Major", Value: "major"},
					},
				}),
				initwiz.NewBoolField(initwiz.BoolFieldOptions{
					Key:         keyUpdatePipeline,
					Title:       "Add update check to CI pipeline?",
					Description: "Add a tfupdate-check job to generated pipelines",
					Default:     false,
				}),
			},
		},
	}
}

// BuildInitConfig builds the update init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
	enabled := keyUpdateEnabled.Get(state)
	if !enabled {
		return nil, nil
	}

	cfg := initConfig{
		Enabled: true,
	}

	if target := keyUpdateTarget.Get(state); target != "" && target != "all" {
		cfg.Target = target
	}
	if bump := keyUpdateBump.Get(state); bump != "" && bump != "minor" {
		cfg.Policy = &initPolicyConfig{Bump: bump}
	}
	if keyUpdatePipeline.Get(state) {
		cfg.Pipeline = true
	}

	return initwiz.NewInitContribution(pluginName, cfg)
}
