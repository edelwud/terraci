package github

import (
	"maps"
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

func TestPlugin_SDKContracts(t *testing.T) {
	p := newContractPlugin()

	t.Run("config", func(t *testing.T) {
		plugintest.AssertBaseConfigPlugin[*configpkg.Config](t, plugintest.BaseConfigPluginContract[*configpkg.Config]{
			Plugin:  p,
			Default: &configpkg.Config{RunsOn: "ubuntu-latest"},
			Configured: &configpkg.Config{
				RunsOn:      "ubuntu-latest",
				Container:   &configpkg.Image{Name: "hashicorp/terraform:1.6", Entrypoint: []string{"/bin/sh"}},
				Env:         map[string]string{"TF_INPUT": "false"},
				Permissions: map[string]string{"id-token": "write"},
				JobDefaults: &configpkg.JobDefaults{
					RunsOn:      "ubuntu-24.04",
					Env:         map[string]string{"DEFAULT": "true"},
					StepsBefore: []configpkg.ConfigStep{{Name: "setup", With: map[string]string{"terraform": "true"}}},
				},
			},
			Decoded: &configpkg.Config{
				RunsOn:      "self-hosted",
				Env:         map[string]string{"DECODED": "true"},
				Permissions: map[string]string{"contents": "read"},
				JobDefaults: &configpkg.JobDefaults{
					RunsOn:     "linux",
					StepsAfter: []configpkg.ConfigStep{{Name: "cleanup", Env: map[string]string{"DONE": "true"}}},
				},
			},
			Mutate: mutateGitHubConfig,
			Equal:  equalGitHubConfig,
		})
	})

	t.Run("preflight", func(t *testing.T) {
		plugintest.AssertPreflightable(t, plugintest.PreflightableContract{
			Plugin:     newContractPlugin(),
			AppContext: plugintest.NewAppContext(t, t.TempDir()),
		})
	})

	t.Run("init contributor", func(t *testing.T) {
		state := initwiz.NewStateMap()
		state.Set("provider", pluginName)
		plugintest.AssertInitContributor(t, plugintest.InitContributorContract{
			Contributor:        newContractPlugin(),
			State:              state,
			ExpectedPluginKey:  pluginName,
			ExpectContribution: true,
			DecodeTarget:       &configpkg.Config{},
		})
	})

	t.Run("ci provider", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")
		p := newContractPlugin()
		plugintest.AssertCIProvider(t, plugintest.CIProviderContract{
			EnvDetector:    p,
			InfoProvider:   p,
			Generator:      p,
			CommentFactory: p,
			AppContext:     plugintest.NewAppContext(t, t.TempDir()),
			IR:             &pipeline.IR{},
			ExpectedName:   pluginName,
			AssertEnv: func(tb testing.TB, detected bool) {
				tb.Helper()
				if !detected {
					tb.Fatal("DetectEnv() = false, want true")
				}
			},
			AssertComment: func(tb testing.TB, service ci.CommentService, ok bool) {
				tb.Helper()
				if !ok || service == nil {
					tb.Fatal("NewCommentService() did not return a service")
				}
			},
		})
	})
}

func newContractPlugin() *Plugin {
	return &Plugin{BasePlugin: plugin.BasePlugin[*configpkg.Config]{
		PluginName: pluginName,
		PluginDesc: "GitHub Actions pipeline generation and PR comments",
		EnableMode: plugin.EnabledWhenConfigured,
		DefaultCfg: func() *configpkg.Config {
			return &configpkg.Config{RunsOn: "ubuntu-latest"}
		},
	}}
}

func mutateGitHubConfig(c *configpkg.Config) {
	if c == nil {
		return
	}
	if c.Container != nil {
		c.Container.Entrypoint = append(c.Container.Entrypoint, "mutated")
	}
	if c.Env == nil {
		c.Env = map[string]string{}
	}
	c.Env["MUTATED"] = "true"
	if c.JobDefaults != nil {
		if c.JobDefaults.Env == nil {
			c.JobDefaults.Env = map[string]string{}
		}
		c.JobDefaults.Env["MUTATED"] = "true"
		if len(c.JobDefaults.StepsBefore) > 0 {
			c.JobDefaults.StepsBefore[0].With["mutated"] = "true"
		}
		if len(c.JobDefaults.StepsAfter) > 0 {
			c.JobDefaults.StepsAfter[0].Env["mutated"] = "true"
		}
	}
}

func equalGitHubConfig(got, want *configpkg.Config) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.RunsOn == want.RunsOn &&
		equalGitHubImage(got.Container, want.Container) &&
		maps.Equal(got.Env, want.Env) &&
		maps.Equal(got.Permissions, want.Permissions) &&
		equalGitHubDefaults(got.JobDefaults, want.JobDefaults)
}

func equalGitHubImage(got, want *configpkg.Image) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Name == want.Name && slices.Equal(got.Entrypoint, want.Entrypoint)
}

func equalGitHubDefaults(got, want *configpkg.JobDefaults) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.RunsOn == want.RunsOn &&
		maps.Equal(got.Env, want.Env) &&
		slices.EqualFunc(got.StepsBefore, want.StepsBefore, equalGitHubStep) &&
		slices.EqualFunc(got.StepsAfter, want.StepsAfter, equalGitHubStep)
}

func equalGitHubStep(got, want configpkg.ConfigStep) bool {
	return got.Name == want.Name &&
		got.Uses == want.Uses &&
		got.Run == want.Run &&
		maps.Equal(got.With, want.With) &&
		maps.Equal(got.Env, want.Env)
}
