package generate

import (
	"maps"
	"strings"

	"github.com/edelwud/terraci/pkg/terraformrun"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

// DefaultBinary is the terraform binary name used when execution.Binary is
// empty. Exported so wizard / tests can refer to a single source of truth.
const (
	DefaultBinary     = "terraform"
	DefaultTofuBinary = "tofu"
)

const (
	defaultTerraformImage = "hashicorp/terraform:1.6"
	defaultTofuImage      = "ghcr.io/opentofu/opentofu:1.6"
)

type settings struct {
	config  *configpkg.Config
	profile terraformrun.Profile
}

func newSettings(cfg *configpkg.Config, profile terraformrun.Profile) settings {
	if cfg == nil {
		cfg = &configpkg.Config{}
	}
	return settings{config: cfg, profile: profile}
}

func (s settings) terraformBinary() string {
	return s.profile.Binary().String()
}

func (s settings) variables() map[string]string {
	variables := make(map[string]string)
	maps.Copy(variables, s.config.Variables)
	return variables
}

func (s settings) defaultImage() configpkg.Image {
	if image := s.config.GetImage(); image != nil && image.Name != "" {
		return *image
	}
	if s.terraformBinary() == DefaultTofuBinary {
		return configpkg.Image{Name: defaultTofuImage}
	}
	return configpkg.Image{Name: defaultTerraformImage}
}

func (s settings) initEnabled() bool {
	return s.profile.InitEnabled()
}

func (s settings) stagesPrefix() string {
	if s.config.StagesPrefix != "" {
		return s.config.StagesPrefix
	}
	return DefaultStagesPrefix
}

func (s settings) workflowRules() []configpkg.Rule {
	return s.config.Rules
}

func (s settings) jobDefaults() *configpkg.JobDefaults {
	return s.config.JobDefaults
}

func (s settings) overwrites() []configpkg.JobOverwrite {
	return s.config.Overwrites
}

func (s settings) cacheEnabled() bool {
	if s.config.Cache != nil && s.config.Cache.Enabled != nil {
		return *s.config.Cache.Enabled
	}
	return true
}

func (s settings) cachePolicy() string {
	if s.config.Cache == nil {
		return ""
	}
	return strings.TrimSpace(s.config.Cache.Policy)
}

func (s settings) cacheKeyTemplate() string {
	if s.config.Cache == nil {
		return ""
	}
	return strings.TrimSpace(s.config.Cache.Key)
}

func (s settings) cachePathTemplates() []string {
	if s.config.Cache == nil {
		return nil
	}
	return s.config.Cache.Paths
}
