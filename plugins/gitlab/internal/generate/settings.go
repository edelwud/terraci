package generate

import (
	"maps"

	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
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

func (s settings) terraformBinary() string {
	if s.config.TerraformBinary != "" {
		return s.config.TerraformBinary
	}
	return "terraform"
}

func (s settings) variables() map[string]string {
	variables := make(map[string]string)
	maps.Copy(variables, s.config.Variables)
	variables["TERRAFORM_BINARY"] = s.terraformBinary()
	return variables
}

func (s settings) defaultImage() configpkg.Image {
	return s.config.GetImage()
}

func (s settings) initEnabled() bool {
	return s.config.InitEnabled
}

func (s settings) planEnabled() bool {
	return s.config.PlanEnabled
}

func (s settings) planOnly() bool {
	return s.config.PlanOnly
}

func (s settings) autoApprove() bool {
	return s.config.AutoApprove
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
	return s.config.CacheEnabled
}

func (s settings) mrCommentEnabled() bool {
	if s.config.MR == nil {
		return false
	}
	if s.config.MR.Comment == nil || s.config.MR.Comment.Enabled == nil {
		return true
	}
	return *s.config.MR.Comment.Enabled
}

func (s settings) summaryJob() summaryJobSettings {
	if s.config.MR == nil || s.config.MR.SummaryJob == nil {
		return summaryJobSettings{}
	}

	cfg := s.config.MR.SummaryJob
	return summaryJobSettings{
		image: cfg.Image,
		tags:  cfg.Tags,
	}
}

type summaryJobSettings struct {
	image *configpkg.Image
	tags  []string
}

func (s summaryJobSettings) configured() bool {
	return s.image != nil || len(s.tags) > 0
}
