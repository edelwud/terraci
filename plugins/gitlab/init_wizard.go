package gitlab

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes GitLab CI fields to the init wizard.

const (
	defaultTerraformImage = "hashicorp/terraform:1.6"
	defaultTofuImage      = "ghcr.io/opentofu/opentofu:1.6"
)

// InitGroups returns the init wizard group specs for GitLab CI.
func (p *Plugin) InitGroups() []*plugin.InitGroupSpec {
	showGitLab := func(s *plugin.StateMap) bool {
		return s.Provider() == "gitlab"
	}

	return []*plugin.InitGroupSpec{
		{
			Title:    "GitLab CI",
			Category: plugin.CategoryProvider,
			Order:    100,
			ShowWhen: showGitLab,
			Fields: []plugin.InitField{
				{
					Key:         "gitlab.image",
					Title:       "Docker Image",
					Description: "Base Docker image for terraform jobs",
					Type:        "string",
					Default:     defaultTerraformImage,
					Placeholder: defaultTerraformImage,
				},
				{
					Key:         "gitlab.stages_prefix",
					Title:       "Stages Prefix",
					Description: "Prefix for pipeline stage names (e.g. deploy-plan-0)",
					Type:        "string",
					Default:     "deploy",
					Placeholder: "deploy",
				},
				{
					Key:         "gitlab.cache_enabled",
					Title:       "Enable .terraform caching?",
					Description: "Cache .terraform directory between pipeline runs",
					Type:        "bool",
					Default:     true,
				},
			},
		},
		{
			Title:    "Pipeline",
			Category: plugin.CategoryPipeline,
			Order:    100,
			ShowWhen: showGitLab,
			Fields: []plugin.InitField{
				{
					Key:         "plan_enabled",
					Title:       "Enable plan stage?",
					Description: "Generate separate plan + apply jobs",
					Type:        "bool",
					Default:     true,
				},
				{
					Key:         "auto_approve",
					Title:       "Auto-approve applies?",
					Description: "Skip manual approval for terraform apply",
					Type:        "bool",
					Default:     false,
				},
			},
		},
	}
}

// BuildInitConfig builds the GitLab CI init contribution.
func (p *Plugin) BuildInitConfig(state *plugin.StateMap) *plugin.InitContribution {
	if state.Provider() != "gitlab" {
		return nil
	}
	binary := state.Binary()
	if binary == "" {
		binary = "terraform"
	}

	image := state.String("gitlab.image")
	if image == "" || image == defaultTerraformImage || image == defaultTofuImage {
		if binary == "tofu" {
			image = defaultTofuImage
		} else {
			image = defaultTerraformImage
		}
	}

	stagesPrefix := state.String("gitlab.stages_prefix")
	if stagesPrefix == "" {
		stagesPrefix = "deploy"
	}

	cacheEnabled := true
	if state.Get("gitlab.cache_enabled") != nil {
		cacheEnabled = state.Bool("gitlab.cache_enabled")
	}

	planEnabled := state.Bool("plan_enabled")
	autoApprove := state.Bool("auto_approve")

	cfg := map[string]any{
		"terraform_binary": binary,
		"image":            map[string]any{"name": image},
		"stages_prefix":    stagesPrefix,
		"plan_enabled":     planEnabled,
		"auto_approve":     autoApprove,
		"cache_enabled":    cacheEnabled,
		"init_enabled":     true,
	}

	// Enable MR comments when summary is enabled
	if state.Bool("summary.enabled") {
		cfg["mr"] = map[string]any{
			"comment": map[string]any{"enabled": true},
		}
	}

	return &plugin.InitContribution{
		PluginKey: "gitlab",
		Config:    cfg,
	}
}
