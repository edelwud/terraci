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

var (
	keyGitlabImage        = initwiz.MustStateKey[string]("gitlab.image")
	keyGitlabStagesPrefix = initwiz.MustStateKey[string]("gitlab.stages_prefix")
	keyGitlabCacheEnabled = initwiz.MustStateKey[bool]("gitlab.cache_enabled")
)

type initConfig struct {
	Image        configpkg.Image `yaml:"image"`
	StagesPrefix string          `yaml:"stages_prefix"`
	CacheEnabled bool            `yaml:"cache_enabled"`
}

// InitGroups returns the init wizard group specs for GitLab CI.
func (p *Plugin) InitGroups() []*initwiz.InitGroupSpec {
	showGitLab := func(s *initwiz.StateMap) bool {
		return initwiz.ProviderKey.Get(s) == pluginName
	}

	return []*initwiz.InitGroupSpec{
		{
			Title:    "GitLab CI",
			Category: initwiz.CategoryProvider,
			Order:    100,
			ShowWhen: showGitLab,
			Fields: []initwiz.InitField{
				initwiz.NewStringField(initwiz.StringFieldOptions{
					Key:         keyGitlabImage,
					Title:       "Docker Image",
					Description: "Base Docker image for terraform jobs",
					Default:     defaultTerraformImage,
					Placeholder: defaultTerraformImage,
				}),
				initwiz.NewStringField(initwiz.StringFieldOptions{
					Key:         keyGitlabStagesPrefix,
					Title:       "Stages Prefix",
					Description: "Prefix for DAG stage names (e.g. deploy-0)",
					Default:     defaultStagesPrefix,
					Placeholder: defaultStagesPrefix,
				}),
				initwiz.NewBoolField(initwiz.BoolFieldOptions{
					Key:         keyGitlabCacheEnabled,
					Title:       "Enable .terraform caching?",
					Description: "Cache .terraform directory between pipeline runs",
					Default:     true,
				}),
			},
		},
		ciplugin.PipelineGroup(pluginName),
	}
}

// BuildInitConfig builds the GitLab CI init contribution.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
	if initwiz.ProviderKey.Get(state) != pluginName {
		return nil, nil
	}
	binary := initwiz.BinaryKey.Get(state)
	if binary == "" {
		binary = "terraform"
	}

	image := keyGitlabImage.Get(state)
	if image == "" || image == defaultTerraformImage || image == defaultTofuImage {
		if binary == "tofu" {
			image = defaultTofuImage
		} else {
			image = defaultTerraformImage
		}
	}

	stagesPrefix := keyGitlabStagesPrefix.Get(state)
	if stagesPrefix == "" {
		stagesPrefix = defaultStagesPrefix
	}

	cacheEnabled := true
	if value, ok := keyGitlabCacheEnabled.Lookup(state); ok {
		cacheEnabled = value
	}

	cfg := initConfig{
		Image:        configpkg.Image{Name: image},
		StagesPrefix: stagesPrefix,
		CacheEnabled: cacheEnabled,
	}

	return initwiz.NewInitContribution(pluginName, cfg)
}
