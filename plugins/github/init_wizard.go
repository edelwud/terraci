package github

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// InitContributor — contributes GitHub Actions fields to the init wizard.

const defaultGitHubRunner = "ubuntu-latest"

// InitGroups returns the init wizard group specs for GitHub Actions.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	showGitHub := func(s *initwiz.StateMap) bool {
		return s.Provider() == "github"
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
		{
			Title:    "Pipeline",
			Category: initwiz.CategoryPipeline,
			Order:    100,
			ShowWhen: showGitHub,
			Fields: []initwiz.InitField{
				{
					Key:         "plan_enabled",
					Title:       "Enable plan stage?",
					Description: "Generate separate plan + apply jobs",
					Type:        initwiz.FieldBool,
					Default:     true,
				},
				{
					Key:         "auto_approve",
					Title:       "Auto-approve applies?",
					Description: "Skip manual approval for terraform apply",
					Type:        initwiz.FieldBool,
					Default:     false,
				},
			},
		},
	}
}

// BuildInitConfig builds the GitHub Actions init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) *initwiz.InitContribution {
	if state.Provider() != "github" {
		return nil
	}
	binary := state.Binary()
	if binary == "" {
		binary = "terraform"
	}

	runsOn := state.String("github.runs_on")
	if runsOn == "" {
		runsOn = defaultGitHubRunner
	}

	planEnabled := state.Bool("plan_enabled")
	autoApprove := state.Bool("auto_approve")

	setupAction := "hashicorp/setup-terraform@v3"
	if binary == "tofu" {
		setupAction = "opentofu/setup-opentofu@v1"
	}

	setupSteps := []map[string]any{
		{"uses": "actions/checkout@v4"},
		{"uses": setupAction},
	}

	cfg := map[string]any{
		"terraform_binary": binary,
		"runs_on":          runsOn,
		"plan_enabled":     planEnabled,
		"auto_approve":     autoApprove,
		"init_enabled":     true,
		"job_defaults": map[string]any{
			"steps_before": setupSteps,
		},
	}

	// Enable PR comments when summary is enabled
	if state.Bool("summary.enabled") {
		cfg["pr"] = map[string]any{
			"comment": map[string]any{"enabled": true},
		}
	}

	return &initwiz.InitContribution{
		PluginKey: "github",
		Config:    cfg,
	}
}
