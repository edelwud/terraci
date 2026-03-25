package github

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes GitHub Actions fields to the init wizard.

const defaultGitHubRunner = "ubuntu-latest"

// InitGroup returns the init wizard group spec for GitHub Actions.
func (p *Plugin) InitGroup() *plugin.InitGroupSpec {
	return &plugin.InitGroupSpec{
		Title: "GitHub Actions",
		Order: 100,
		Fields: []plugin.InitField{
			{
				Key:         "github.runs_on",
				Title:       "Runner Label",
				Description: "GitHub Actions runs-on value",
				Type:        "string",
				Default:     defaultGitHubRunner,
				Placeholder: defaultGitHubRunner,
			},
		},
		ShowWhen: func(s plugin.InitState) bool {
			return s.Provider() == "github"
		},
	}
}

// BuildInitConfig builds the GitHub Actions init contribution.
func (p *Plugin) BuildInitConfig(state plugin.InitState) *plugin.InitContribution {
	if state.Provider() != "github" {
		return nil
	}
	binary := state.Binary()
	if binary == "" {
		binary = "terraform"
	}

	runsOn, _ := state.Get("github.runs_on").(string) //nolint:errcheck // safe type assertion
	if runsOn == "" {
		runsOn = defaultGitHubRunner
	}

	planEnabled, _ := state.Get("plan_enabled").(bool) //nolint:errcheck // safe type assertion
	autoApprove, _ := state.Get("auto_approve").(bool) //nolint:errcheck // safe type assertion
	enableMR, _ := state.Get("enable_mr").(bool)       //nolint:errcheck // safe type assertion

	setupAction := "hashicorp/setup-terraform@v3"
	if binary == "tofu" {
		setupAction = "opentofu/setup-opentofu@v1"
	}

	m := map[string]any{
		"terraform_binary": binary,
		"runs_on":          runsOn,
		"plan_enabled":     planEnabled,
		"auto_approve":     autoApprove,
		"init_enabled":     true,
		"job_defaults": map[string]any{
			"steps_before": []map[string]any{
				{"uses": "actions/checkout@v4"},
				{"uses": setupAction},
			},
		},
	}

	if enableMR {
		m["permissions"] = map[string]string{
			"contents":      "read",
			"pull-requests": "write",
		}
		m["pr"] = map[string]any{
			"comment": map[string]any{},
		}
	}

	return &plugin.InitContribution{PluginKey: "github", Config: m}
}
