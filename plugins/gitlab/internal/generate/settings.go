package generate

import (
	"maps"
	"strings"

	"github.com/edelwud/terraci/pkg/execution"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

// DefaultBinary is the terraform binary name used when execution.Binary is
// empty. Exported so wizard / tests can refer to a single source of truth.
const DefaultBinary = "terraform"

type settings struct {
	config    *configpkg.Config
	execution execution.Config
}

func newSettings(cfg *configpkg.Config, execCfg execution.Config) settings {
	if cfg == nil {
		cfg = &configpkg.Config{}
	}
	return settings{config: cfg, execution: execCfg}
}

func (s settings) terraformBinary() string {
	if s.execution.Binary != "" {
		return s.execution.Binary
	}
	return DefaultBinary
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
	return s.execution.InitEnabled
}

func (s settings) planEnabled() bool {
	return s.execution.PlanEnabled
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
	if s.config.Cache != nil && s.config.Cache.Enabled != nil {
		return *s.config.Cache.Enabled
	}
	return s.config.CacheEnabled
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
