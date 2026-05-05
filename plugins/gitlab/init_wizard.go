package gitlab

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// InitContributor — contributes GitLab CI fields to the init wizard.

const (
	defaultTerraformImage = "hashicorp/terraform:1.6"
	defaultTofuImage      = "ghcr.io/opentofu/opentofu:1.6"
)

// Wizard StateMap keys. Centralized so InitGroups field definitions and
// BuildInitConfig consumers can never drift apart on a typo. The "auto_approve"
// and "summary.enabled" keys are owned by other groups (pipeline category and
// summary plugin); we reference them but don't define them here.
const (
	keyGitlabImage        = "gitlab.image"
	keyGitlabStagesPrefix = "gitlab.stages_prefix"
	keyGitlabCacheEnabled = "gitlab.cache_enabled"
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
					Key:         keyGitlabImage,
					Title:       "Docker Image",
					Description: "Base Docker image for terraform jobs",
					Type:        initwiz.FieldString,
					Default:     defaultTerraformImage,
					Placeholder: defaultTerraformImage,
				},
				{
					Key:         keyGitlabStagesPrefix,
					Title:       "Stages Prefix",
					Description: "Prefix for pipeline stage names (e.g. deploy-plan-0)",
					Type:        initwiz.FieldString,
					Default:     "deploy",
					Placeholder: "deploy",
				},
				{
					Key:         keyGitlabCacheEnabled,
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

	image := state.String(keyGitlabImage)
	if image == "" || image == defaultTerraformImage || image == defaultTofuImage {
		if binary == "tofu" {
			image = defaultTofuImage
		} else {
			image = defaultTerraformImage
		}
	}

	stagesPrefix := state.String(keyGitlabStagesPrefix)
	if stagesPrefix == "" {
		stagesPrefix = "deploy"
	}

	cacheEnabled := true
	if state.Get(keyGitlabCacheEnabled) != nil {
		cacheEnabled = state.Bool(keyGitlabCacheEnabled)
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
