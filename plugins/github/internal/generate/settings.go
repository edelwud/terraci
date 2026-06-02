package generate

import (
	"maps"

	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
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
			RunsOn: "ubuntu-latest",
		}
	}
	return s.config
}

func (s settings) env() map[string]string {
	env := make(map[string]string)
	maps.Copy(env, s.configOrDefault().Env)
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
