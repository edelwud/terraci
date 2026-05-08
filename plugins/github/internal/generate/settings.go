package generate

import (
	"maps"

	"github.com/edelwud/terraci/pkg/execution"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

type settings struct {
	config    *configpkg.Config
	execution execution.Config
}

func newSettings(cfg *configpkg.Config, execCfg execution.Config) settings {
	return settings{config: cfg, execution: execCfg}
}

func (s settings) configOrDefault() *configpkg.Config {
	if s.config == nil {
		return &configpkg.Config{
			RunsOn: "ubuntu-latest",
		}
	}
	return s.config
}

func (s settings) terraformBinary() string {
	if s.execution.Binary != "" {
		return s.execution.Binary
	}
	return "terraform"
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
	return s.execution.PlanEnabled
}

func (s settings) planOnly() bool {
	return s.configOrDefault().PlanOnly
}

func (s settings) initEnabled() bool {
	return s.execution.InitEnabled
}
