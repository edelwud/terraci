package gitlab

import (
	"maps"
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline/pipelinetest"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func TestPlugin_SDKContracts(t *testing.T) {
	p := newContractPlugin()

	t.Run("config", func(t *testing.T) {
		configuredImage := configpkg.Image{Name: "hashicorp/terraform:1.7", Entrypoint: []string{"/bin/sh"}}
		decodedImage := configpkg.Image{Name: "custom/terraform:latest"}
		cacheEnabled := true
		plugintest.AssertBaseConfigPlugin[*configpkg.Config](t, plugintest.BaseConfigPluginContract[*configpkg.Config]{
			Plugin: p,
			Default: &configpkg.Config{
				StagesPrefix: defaultStagesPrefix,
				Cache:        &configpkg.CacheConfig{Enabled: &cacheEnabled},
			},
			Configured: &configpkg.Config{
				Image:        &configuredImage,
				StagesPrefix: "deploy",
				Variables:    map[string]string{"TF_INPUT": "false"},
				Cache:        &configpkg.CacheConfig{Paths: []string{".terraform"}, Policy: "pull-push"},
				Rules:        []configpkg.Rule{{If: "$CI_PIPELINE_SOURCE", Changes: []string{"**/*.tf"}}},
			},
			Decoded: &configpkg.Config{
				Image:        &decodedImage,
				StagesPrefix: "decoded",
				Variables:    map[string]string{"DECODED": "true"},
				Cache:        &configpkg.CacheConfig{Paths: []string{"decoded"}, Policy: "pull"},
				Rules:        []configpkg.Rule{{If: "$CI_COMMIT_BRANCH", Changes: []string{"decoded/**"}}},
			},
			Mutate: mutateGitLabConfig,
			Equal:  equalGitLabConfig,
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
		initwiz.ProviderKey.Set(state, pluginName)
		plugintest.AssertInitContributor(t, plugintest.InitContributorContract{
			Contributor:        newContractPlugin(),
			State:              state,
			ExpectedPluginKey:  pluginName,
			ExpectContribution: true,
			DecodeTarget:       &configpkg.Config{},
		})
	})

	t.Run("ci provider", func(t *testing.T) {
		t.Setenv("GITLAB_CI", "true")
		p := newContractPlugin()
		plugintest.AssertCIProvider(t, plugintest.CIProviderContract{
			EnvDetector:    p,
			InfoProvider:   p,
			Generator:      p,
			CommentFactory: p,
			AppContext:     plugintest.NewAppContext(t, t.TempDir()),
			IR:             pipelinetest.MustCommandIR(t),
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
		PluginDesc: "GitLab CI pipeline generation and MR comments",
		EnableMode: plugin.EnabledWhenConfigured,
		DefaultCfg: func() *configpkg.Config {
			cacheEnabled := true
			return &configpkg.Config{
				StagesPrefix: defaultStagesPrefix,
				Cache:        &configpkg.CacheConfig{Enabled: &cacheEnabled},
			}
		},
	}}
}

func mutateGitLabConfig(c *configpkg.Config) {
	if c == nil {
		return
	}
	if c.Image != nil {
		c.Image.Entrypoint = append(c.Image.Entrypoint, "mutated")
	}
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
	return equalGitLabImage(got.Image, want.Image) &&
		got.StagesPrefix == want.StagesPrefix &&
		maps.Equal(got.Variables, want.Variables) &&
		equalGitLabCache(got.Cache, want.Cache) &&
		slices.EqualFunc(got.Rules, want.Rules, equalGitLabRule)
}

func equalGitLabImage(got, want *configpkg.Image) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Name == want.Name && slices.Equal(got.Entrypoint, want.Entrypoint)
}

func equalGitLabCache(got, want *configpkg.CacheConfig) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Policy == want.Policy &&
		got.Key == want.Key &&
		equalBoolPointer(got.Enabled, want.Enabled) &&
		slices.Equal(got.Paths, want.Paths)
}

func equalBoolPointer(got, want *bool) bool {
	if got == nil || want == nil {
		return got == want
	}
	return *got == *want
}

func equalGitLabRule(got, want configpkg.Rule) bool {
	return got.If == want.If &&
		got.When == want.When &&
		slices.Equal(got.Changes, want.Changes)
}
