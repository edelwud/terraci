package gitlab

import (
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	"github.com/edelwud/terraci/plugins/internal/ciplugin"
)

// InitContributor — contributes GitLab CI fields to the init wizard.

const (
	defaultTerraformImage = "hashicorp/terraform:1.6"
	defaultTofuImage      = "ghcr.io/opentofu/opentofu:1.6"
)

// Wizard StateMap keys. Centralized so InitGroups field definitions and
// BuildInitConfig consumers can never drift apart on a typo.
const (
	keyGitlabImage        = "gitlab.image"
	keyGitlabStagesPrefix = "gitlab.stages_prefix"
	keyGitlabCacheEnabled = "gitlab.cache_enabled"
)

type initConfig struct {
	Image        configpkg.Image `yaml:"image"`
	StagesPrefix string          `yaml:"stages_prefix"`
	CacheEnabled bool            `yaml:"cache_enabled"`
}

// InitGroups returns the init wizard group specs for GitLab CI.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	showGitLab := func(s *initwiz.StateMap) bool {
		return s.Provider() == pluginName
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
					Description: "Prefix for DAG stage names (e.g. deploy-0)",
					Type:        initwiz.FieldString,
					Default:     defaultStagesPrefix,
					Placeholder: defaultStagesPrefix,
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
		ciplugin.PipelineGroup(pluginName),
	}
}

// BuildInitConfig builds the GitLab CI init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
	if state.Provider() != pluginName {
		return nil, nil
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
		stagesPrefix = defaultStagesPrefix
	}

	cacheEnabled := true
	if state.Get(keyGitlabCacheEnabled) != nil {
		cacheEnabled = state.Bool(keyGitlabCacheEnabled)
	}

	cfg := initConfig{
		Image:        configpkg.Image{Name: image},
		StagesPrefix: stagesPrefix,
		CacheEnabled: cacheEnabled,
	}

	return initwiz.NewInitContribution(pluginName, cfg)
}
