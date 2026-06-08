package gitlab

import (
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

// InitContributor — contributes GitLab CI fields to the init wizard.

const (
	defaultTerraformImage = "hashicorp/terraform:1.6"
	defaultTofuImage      = "ghcr.io/opentofu/opentofu:1.6"
)

var (
	initConfigKey         = config.MustExtensionKey(pluginName)
	keyGitlabImage        = initwiz.MustStateKey[string]("gitlab.image")
	keyGitlabStagesPrefix = initwiz.MustStateKey[string]("gitlab.stages_prefix")
	keyGitlabCacheEnabled = initwiz.MustStateKey[bool]("gitlab.cache.enabled")
)

type initConfig struct {
	Image        *configpkg.Image       `yaml:"image,omitempty"`
	StagesPrefix string                 `yaml:"stages_prefix"`
	Cache        *configpkg.CacheConfig `yaml:"cache,omitempty"`
}

// InitGroups returns the init wizard group specs for GitLab CI.
func (p *Plugin) InitGroups() ([]initwiz.InitGroup, error) {
	showGitLab := func(s *initwiz.StateMap) bool {
		return initwiz.ProviderKey.Get(s) == pluginName
	}

	image, err := initwiz.NewStringField(initwiz.StringFieldOptions{
		Key:         keyGitlabImage,
		Title:       "Docker Image",
		Description: "Base Docker image for terraform jobs",
		Default:     defaultTerraformImage,
		Placeholder: defaultTerraformImage,
	})
	if err != nil {
		return nil, err
	}
	stagesPrefix, err := initwiz.NewStringField(initwiz.StringFieldOptions{
		Key:         keyGitlabStagesPrefix,
		Title:       "Stages Prefix",
		Description: "Prefix for DAG stage names (e.g. deploy-0)",
		Default:     defaultStagesPrefix,
		Placeholder: defaultStagesPrefix,
	})
	if err != nil {
		return nil, err
	}
	cache, err := initwiz.NewBoolField(initwiz.BoolFieldOptions{
		Key:         keyGitlabCacheEnabled,
		Title:       "Enable .terraform caching?",
		Description: "Cache .terraform directory between pipeline runs",
		Default:     true,
	})
	if err != nil {
		return nil, err
	}
	group, err := initwiz.NewInitGroup(initwiz.InitGroupOptions{
		Title:    "GitLab CI",
		Category: initwiz.CategoryProvider,
		Order:    100,
		ShowWhen: showGitLab,
		Fields:   []initwiz.InitField{image, stagesPrefix, cache},
	})
	if err != nil {
		return nil, err
	}
	return []initwiz.InitGroup{group}, nil
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
	defaultImage := defaultTerraformImage
	if binary == "tofu" {
		defaultImage = defaultTofuImage
	}
	var imageOverride *configpkg.Image
	if image != "" && image != defaultImage {
		imageOverride = &configpkg.Image{Name: image}
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
		Image:        imageOverride,
		StagesPrefix: stagesPrefix,
		Cache:        &configpkg.CacheConfig{Enabled: &cacheEnabled},
	}

	return initwiz.NewInitContribution(initConfigKey, cfg)
}
