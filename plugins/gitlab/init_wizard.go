package gitlab

import "github.com/edelwud/terraci/pkg/plugin"

// InitContributor — contributes GitLab CI fields to the init wizard.

const (
	defaultTerraformImage = "hashicorp/terraform:1.6"
	defaultTofuImage      = "ghcr.io/opentofu/opentofu:1.6"
)

// InitGroup returns the init wizard group spec for GitLab CI.
func (p *Plugin) InitGroup() *plugin.InitGroupSpec {
	return &plugin.InitGroupSpec{
		Title: "GitLab CI",
		Order: 100,
		Fields: []plugin.InitField{
			{
				Key:         "gitlab.image",
				Title:       "Docker Image",
				Description: "Base Docker image for terraform jobs",
				Type:        "string",
				Default:     defaultTerraformImage,
				Placeholder: defaultTerraformImage,
			},
		},
		ShowWhen: func(s plugin.InitState) bool {
			return s.Provider() == "gitlab"
		},
	}
}

// BuildInitConfig builds the GitLab CI init contribution.
func (p *Plugin) BuildInitConfig(state plugin.InitState) *plugin.InitContribution {
	if state.Provider() != "gitlab" {
		return nil
	}
	binary := state.Binary()
	if binary == "" {
		binary = "terraform"
	}

	image, _ := state.Get("gitlab.image").(string) //nolint:errcheck // safe type assertion
	if image == "" {
		if binary == "tofu" {
			image = defaultTofuImage
		} else {
			image = defaultTerraformImage
		}
	}

	planEnabled, _ := state.Get("plan_enabled").(bool) //nolint:errcheck // safe type assertion
	autoApprove, _ := state.Get("auto_approve").(bool) //nolint:errcheck // safe type assertion

	return &plugin.InitContribution{
		PluginKey: "gitlab",
		Config: map[string]any{
			"terraform_binary": binary,
			"image":            map[string]any{"name": image},
			"plan_enabled":     planEnabled,
			"auto_approve":     autoApprove,
			"init_enabled":     true,
		},
	}
}
