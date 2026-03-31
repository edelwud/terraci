package generate

import (
	"maps"

	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
	domainpkg "github.com/edelwud/terraci/plugins/github/internal/domain"
)

type settings struct {
	config *configpkg.Config
}

func newSettings(cfg *configpkg.Config) settings {
	return settings{config: cfg}
}

func (s settings) configOrDefault() *configpkg.Config {
	if s.config == nil {
		return &configpkg.Config{
			TerraformBinary: "terraform",
			RunsOn:          "ubuntu-latest",
			PlanEnabled:     true,
			InitEnabled:     true,
		}
	}
	return s.config
}

func (s settings) terraformBinary() string {
	if binary := s.configOrDefault().TerraformBinary; binary != "" {
		return binary
	}
	return "terraform"
}

func (s settings) runsOn() string {
	cfg := s.configOrDefault()
	if cfg.JobDefaults != nil && cfg.JobDefaults.RunsOn != "" {
		return cfg.JobDefaults.RunsOn
	}
	if cfg.RunsOn != "" {
		return cfg.RunsOn
	}
	return "ubuntu-latest"
}

func (s settings) container() *domainpkg.Container {
	cfg := s.configOrDefault()
	if cfg.JobDefaults != nil && cfg.JobDefaults.Container != nil {
		return &domainpkg.Container{Image: cfg.JobDefaults.Container.Name}
	}
	if cfg.Container != nil {
		return &domainpkg.Container{Image: cfg.Container.Name}
	}
	return nil
}

func (s settings) env() map[string]string {
	env := make(map[string]string)
	maps.Copy(env, s.configOrDefault().Env)
	env["TERRAFORM_BINARY"] = s.terraformBinary()
	return env
}

func (s settings) permissions() map[string]string {
	cfg := s.configOrDefault()
	if len(cfg.Permissions) != 0 {
		return cfg.Permissions
	}
	return map[string]string{
		"contents":      "read",
		"pull-requests": "write",
	}
}

func (s settings) planEnabled() bool {
	return s.configOrDefault().PlanEnabled
}

func (s settings) planOnly() bool {
	return s.configOrDefault().PlanOnly
}

func (s settings) autoApprove() bool {
	return s.configOrDefault().AutoApprove
}

func (s settings) initEnabled() bool {
	return s.configOrDefault().InitEnabled
}

func (s settings) prEnabled() bool {
	cfg := s.config
	if cfg == nil || cfg.PR == nil {
		return false
	}
	if cfg.PR.Comment == nil || cfg.PR.Comment.Enabled == nil {
		return true
	}
	return *cfg.PR.Comment.Enabled
}

func (s settings) summaryRunsOn() string {
	cfg := s.config
	if cfg == nil || cfg.PR == nil || cfg.PR.SummaryJob == nil {
		return ""
	}
	return cfg.PR.SummaryJob.RunsOn
}

func (s settings) stepsBefore(jobType configpkg.JobOverwriteType) []domainpkg.Step {
	return stepsBefore(s.config, jobType)
}

func (s settings) stepsAfter(jobType configpkg.JobOverwriteType) []domainpkg.Step {
	return stepsAfter(s.config, jobType)
}
