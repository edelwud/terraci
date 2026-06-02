package github

import (
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

// InitContributor — contributes GitHub Actions fields to the init wizard.

const defaultGitHubRunner = "ubuntu-latest"

var keyGitHubRunsOn = initwiz.MustStateKey[string]("github.runs_on")

type initConfig struct {
	RunsOn      string                 `yaml:"runs_on"`
	JobDefaults *configpkg.JobDefaults `yaml:"job_defaults,omitempty"`
}

// InitGroups returns the init wizard group specs for GitHub Actions.
func (p *Plugin) InitGroups() ([]initwiz.InitGroup, error) {
	showGitHub := func(s *initwiz.StateMap) bool {
		return initwiz.ProviderKey.Get(s) == pluginName
	}

	runsOn, err := initwiz.NewStringField(initwiz.StringFieldOptions{
		Key:         keyGitHubRunsOn,
		Title:       "Runner Label",
		Description: "GitHub Actions runs-on value",
		Default:     defaultGitHubRunner,
		Placeholder: defaultGitHubRunner,
	})
	if err != nil {
		return nil, err
	}
	group, err := initwiz.NewInitGroup(initwiz.InitGroupOptions{
		Title:    "GitHub Actions",
		Category: initwiz.CategoryProvider,
		Order:    100,
		ShowWhen: showGitHub,
		Fields:   []initwiz.InitField{runsOn},
	})
	if err != nil {
		return nil, err
	}
	return []initwiz.InitGroup{group}, nil
}

// BuildInitConfig builds the GitHub Actions init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
	if initwiz.ProviderKey.Get(state) != pluginName {
		return nil, nil
	}
	binary := initwiz.BinaryKey.Get(state)
	if binary == "" {
		binary = "terraform"
	}

	runsOn := keyGitHubRunsOn.Get(state)
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
