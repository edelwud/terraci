package github

import (
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
	"github.com/edelwud/terraci/plugins/internal/ciplugin"
)

// InitContributor — contributes GitHub Actions fields to the init wizard.

const defaultGitHubRunner = "ubuntu-latest"

type initConfig struct {
	RunsOn      string                 `yaml:"runs_on"`
	JobDefaults *configpkg.JobDefaults `yaml:"job_defaults,omitempty"`
}

// InitGroups returns the init wizard group specs for GitHub Actions.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	showGitHub := func(s *initwiz.StateMap) bool {
		return s.Provider() == pluginName
	}

	return []*initwiz.InitGroupSpec{
		{
			Title:    "GitHub Actions",
			Category: initwiz.CategoryProvider,
			Order:    100,
			ShowWhen: showGitHub,
			Fields: []initwiz.InitField{
				{
					Key:         "github.runs_on",
					Title:       "Runner Label",
					Description: "GitHub Actions runs-on value",
					Type:        initwiz.FieldString,
					Default:     defaultGitHubRunner,
					Placeholder: defaultGitHubRunner,
				},
			},
		},
		ciplugin.PipelineGroup(pluginName),
	}
}

// BuildInitConfig builds the GitHub Actions init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
	if state.Provider() != pluginName {
		return nil, nil
	}
	binary := state.Binary()
	if binary == "" {
		binary = "terraform"
	}

	runsOn := state.String("github.runs_on")
	if runsOn == "" {
		runsOn = defaultGitHubRunner
	}

	setupAction := "hashicorp/setup-terraform@v3"
	if binary == "tofu" {
		setupAction = "opentofu/setup-opentofu@v1"
	}

	setupSteps := []configpkg.ConfigStep{
		{Uses: "actions/checkout@v4"},
		{Uses: setupAction},
	}

	cfg := initConfig{
		RunsOn: runsOn,
		JobDefaults: &configpkg.JobDefaults{
			StepsBefore: setupSteps,
		},
	}

	return initwiz.NewInitContribution(pluginName, cfg)
}
