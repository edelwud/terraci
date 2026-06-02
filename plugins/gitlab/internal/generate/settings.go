package generate

import (
	"errors"
	"maps"
	"strings"

	"github.com/edelwud/terraci/pkg/pipeline"
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
	config *configpkg.Config
}

func newSettings(cfg *configpkg.Config) settings {
	if cfg == nil {
		cfg = &configpkg.Config{}
	}
	return settings{config: cfg}
}

func (s settings) variables() map[string]string {
	variables := make(map[string]string)
	maps.Copy(variables, s.config.Variables)
	return variables
}

func (s settings) defaultImage(ir *pipeline.IR) (configpkg.Image, error) {
	if image := s.config.GetImage(); image != nil && image.Name != "" {
		return *image, nil
	}
	runtime, ok := ir.TerraformRuntime()
	if !ok {
		return configpkg.Image{Name: defaultTerraformImage}, nil
	}
	if runtime.Mixed() {
		return configpkg.Image{}, errors.New("pipeline IR contains mixed Terraform binaries; configure extensions.gitlab.image explicitly to choose a CI image")
	}
	if runtime.Binary() == DefaultTofuBinary {
		return configpkg.Image{Name: defaultTofuImage}, nil
	}
	return configpkg.Image{Name: defaultTerraformImage}, nil
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
