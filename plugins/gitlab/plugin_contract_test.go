package gitlab

import (
	"maps"
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func TestPlugin_BaseConfigContract(t *testing.T) {
	p := &Plugin{BasePlugin: plugin.BasePlugin[*configpkg.Config]{
		PluginName: pluginName,
		PluginDesc: "GitLab CI pipeline generation and MR comments",
		EnableMode: plugin.EnabledWhenConfigured,
		DefaultCfg: func() *configpkg.Config {
			return &configpkg.Config{
				Image:        configpkg.Image{Name: "hashicorp/terraform:1.6"},
				StagesPrefix: defaultStagesPrefix,
				CacheEnabled: true,
			}
		},
	}}

	plugintest.AssertBaseConfigPlugin[*configpkg.Config](t, plugintest.BaseConfigPluginContract[*configpkg.Config]{
		Plugin: p,
		Default: &configpkg.Config{
			Image:        configpkg.Image{Name: "hashicorp/terraform:1.6"},
			StagesPrefix: defaultStagesPrefix,
			CacheEnabled: true,
		},
		Configured: &configpkg.Config{
			Image:        configpkg.Image{Name: "hashicorp/terraform:1.7", Entrypoint: []string{"/bin/sh"}},
			StagesPrefix: "deploy",
			Variables:    map[string]string{"TF_INPUT": "false"},
			CacheEnabled: true,
			Cache:        &configpkg.CacheConfig{Paths: []string{".terraform"}, Policy: "pull-push"},
			Rules:        []configpkg.Rule{{If: "$CI_PIPELINE_SOURCE", Changes: []string{"**/*.tf"}}},
		},
		Decoded: &configpkg.Config{
			Image:        configpkg.Image{Name: "custom/terraform:latest"},
			StagesPrefix: "decoded",
			Variables:    map[string]string{"DECODED": "true"},
			Cache:        &configpkg.CacheConfig{Paths: []string{"decoded"}, Policy: "pull"},
			Rules:        []configpkg.Rule{{If: "$CI_COMMIT_BRANCH", Changes: []string{"decoded/**"}}},
		},
		Mutate: mutateGitLabConfig,
		Equal:  equalGitLabConfig,
	})
}

func mutateGitLabConfig(c *configpkg.Config) {
	if c == nil {
		return
	}
	c.Image.Entrypoint = append(c.Image.Entrypoint, "mutated")
	if c.Variables == nil {
		c.Variables = map[string]string{}
	}
	c.Variables["MUTATED"] = "true"
	if c.Cache != nil {
		c.Cache.Paths = append(c.Cache.Paths, "mutated")
	}
	if len(c.Rules) > 0 {
		c.Rules[0].Changes = append(c.Rules[0].Changes, "mutated")
	}
}

func equalGitLabConfig(got, want *configpkg.Config) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Image.Name == want.Image.Name &&
		slices.Equal(got.Image.Entrypoint, want.Image.Entrypoint) &&
		got.StagesPrefix == want.StagesPrefix &&
		got.CacheEnabled == want.CacheEnabled &&
		maps.Equal(got.Variables, want.Variables) &&
		equalGitLabCache(got.Cache, want.Cache) &&
		slices.EqualFunc(got.Rules, want.Rules, equalGitLabRule)
}

func equalGitLabCache(got, want *configpkg.CacheConfig) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Policy == want.Policy &&
		got.Key == want.Key &&
		slices.Equal(got.Paths, want.Paths)
}

func equalGitLabRule(got, want configpkg.Rule) bool {
	return got.If == want.If &&
		got.When == want.When &&
		slices.Equal(got.Changes, want.Changes)
}
