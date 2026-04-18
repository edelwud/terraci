package gitlab

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// InitContributor — contributes GitLab CI fields to the init wizard.

const (
	defaultTerraformImage = "hashicorp/terraform:1.6"
	defaultTofuImage      = "ghcr.io/opentofu/opentofu:1.6"
)

// InitGroups returns the init wizard group specs for GitLab CI.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	showGitLab := func(s *initwiz.StateMap) bool {
		return s.Provider() == "gitlab"
	}

	return []*initwiz.InitGroupSpec{
		{
			Title:    "GitLab CI",
			Category: initwiz.CategoryProvider,
			Order:    100,
			ShowWhen: showGitLab,
			Fields: []initwiz.InitField{
				{
					Key:         "gitlab.image",
					Title:       "Docker Image",
					Description: "Base Docker image for terraform jobs",
					Type:        initwiz.FieldString,
					Default:     defaultTerraformImage,
					Placeholder: defaultTerraformImage,
				},
				{
					Key:         "gitlab.stages_prefix",
					Title:       "Stages Prefix",
					Description: "Prefix for pipeline stage names (e.g. deploy-plan-0)",
					Type:        initwiz.FieldString,
					Default:     "deploy",
					Placeholder: "deploy",
				},
				{
					Key:         "gitlab.cache_enabled",
					Title:       "Enable .terraform caching?",
					Description: "Cache .terraform directory between pipeline runs",
					Type:        initwiz.FieldBool,
					Default:     true,
				},
			},
		},
		{
			Title:    "Pipeline",
			Category: initwiz.CategoryPipeline,
			Order:    100,
			ShowWhen: showGitLab,
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

// BuildInitConfig builds the GitLab CI init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) *initwiz.InitContribution {
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

	autoApprove := state.Bool("auto_approve")

	cfg := map[string]any{
		"image":         map[string]any{"name": image},
		"stages_prefix": stagesPrefix,
		"auto_approve":  autoApprove,
		"cache_enabled": cacheEnabled,
	}

	// Enable MR comments when summary is enabled
	if state.Bool("summary.enabled") {
		cfg["mr"] = map[string]any{
			"comment": map[string]any{"enabled": true},
		}
	}

	return &initwiz.InitContribution{
		PluginKey: "gitlab",
		Config:    cfg,
	}
}
