package plugintest

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/cache/blobcache/blobtest"
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/pipelinetest"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/workflow"
)

func TestAssertBaseConfigPlugin(t *testing.T) {
	p := &contractPlugin{BasePlugin: plugin.BasePlugin[*contractConfig]{
		PluginName: "contract",
		PluginDesc: "contract plugin",
		EnableMode: plugin.EnabledExplicitly,
		DefaultCfg: func() *contractConfig {
			return &contractConfig{
				Enabled: true,
				Labels:  []string{"default"},
				Nested:  &contractNested{Name: "default"},
			}
		},
		IsEnabledFn: func(c *contractConfig) bool { return c != nil && c.Enabled },
	}}

	AssertBaseConfigPlugin[*contractConfig](t, BaseConfigPluginContract[*contractConfig]{
		Plugin: p,
		Default: &contractConfig{
			Enabled: true,
			Labels:  []string{"default"},
			Nested:  &contractNested{Name: "default"},
		},
		Configured: &contractConfig{
			Enabled: true,
			Labels:  []string{"configured"},
			Nested:  &contractNested{Name: "configured"},
		},
		Decoded: &contractConfig{
			Enabled: true,
			Labels:  []string{"decoded"},
			Nested:  &contractNested{Name: "decoded"},
		},
		Mutate: mutateContractConfig,
		Equal:  equalContractConfig,
	})
}

func TestAssertCommandBinding(t *testing.T) {
	p := &contractPlugin{BasePlugin: plugin.BasePlugin[*contractConfig]{
		PluginName: "contract",
		PluginDesc: "contract plugin",
		EnableMode: plugin.EnabledAlways,
		DefaultCfg: func() *contractConfig { return &contractConfig{} },
	}}

	AssertCommandBinding[*contractPlugin](t, CommandBindingContract[*contractPlugin]{
		Name:   "contract",
		Plugin: p,
		AssertResolved: func(tb testing.TB, got *contractPlugin) {
			tb.Helper()
			if got != p {
				tb.Fatalf("resolved plugin = %p, want %p", got, p)
			}
		},
	})
}

func TestAssertRequireEnabled(t *testing.T) {
	AssertRequireEnabled(t, RequireEnabledContract{
		Enabled:  staticEnabled(true),
		Disabled: staticEnabled(false),
		Message:  "contract plugin is disabled",
	})
}

func TestAssertRuntimeProvider(t *testing.T) {
	p := &contractRuntimePlugin{contractPlugin: contractPlugin{BasePlugin: plugin.BasePlugin[*contractConfig]{
		PluginName: "contract",
		PluginDesc: "contract plugin",
		EnableMode: plugin.EnabledAlways,
		DefaultCfg: func() *contractConfig { return &contractConfig{} },
	}}}

	AssertRuntimeProvider[*contractRuntime](t, RuntimeProviderContract[*contractRuntime]{
		Provider: p,
		AssertRuntime: func(tb testing.TB, got *contractRuntime) {
			tb.Helper()
			if got == nil || got.Name != "contract" {
				tb.Fatalf("runtime = %#v, want named runtime", got)
			}
		},
	})
}

func TestAssertPipelineContributor(t *testing.T) {
	AssertPipelineContributor(t, PipelineContributorContract{
		Contributor:      contractContributor{},
		ExpectedJobNames: []string{"first", "second"},
	})
}

func TestAssertPreflightable(t *testing.T) {
	AssertPreflightable(t, PreflightableContract{
		Plugin: contractPreflightPlugin{},
	})
}

func TestAssertInitContributor(t *testing.T) {
	AssertInitContributor(t, InitContributorContract{
		Contributor:        contractInitContributor{},
		ExpectedPluginKey:  "contract",
		ExpectContribution: true,
		DecodeTarget:       &contractInitConfig{},
	})
}

func TestAssertVersionProvider(t *testing.T) {
	AssertVersionProvider(t, VersionProviderContract{
		Provider:     contractVersionProvider{},
		ExpectedKeys: []string{"engine"},
	})
}

func TestAssertKVCacheProvider(t *testing.T) {
	AssertKVCacheProvider(t, KVCacheProviderContract{
		Provider: contractKVCacheProvider{},
		Value:    []byte("cached"),
	})
}

func TestAssertBlobStoreProvider(t *testing.T) {
	AssertBlobStoreProvider(t, BlobStoreProviderContract{
		Provider: contractBlobStoreProvider{},
		Value:    []byte("blobbed"),
	})
}

func TestAssertChangeDetector(t *testing.T) {
	AssertChangeDetector(t, ChangeDetectorContract{
		Detector: contractChangeDetector{},
		AssertResult: func(tb testing.TB, got *workflow.ChangeDetectionResult) {
			tb.Helper()
			if !slices.Equal(got.Files, []string{"changed.tf"}) {
				tb.Fatalf("changed files = %v, want [changed.tf]", got.Files)
			}
		},
	})
}

func TestAssertCIProvider(t *testing.T) {
	provider := contractCIProvider{}
	AssertCIProvider(t, CIProviderContract{
		EnvDetector:    provider,
		InfoProvider:   provider,
		Generator:      provider,
		CommentFactory: provider,
		IR:             pipelinetest.MustCommandIR(t),
		ExpectedName:   "contract-ci",
		AssertEnv: func(tb testing.TB, detected bool) {
			tb.Helper()
			if !detected {
				tb.Fatal("DetectEnv() = false, want true")
			}
		},
		AssertComment: func(tb testing.TB, service ci.CommentService, ok bool) {
			tb.Helper()
			if !ok || service == nil {
				tb.Fatal("NewCommentService() did not return service")
			}
		},
	})
}

type contractPlugin struct {
	plugin.BasePlugin[*contractConfig]
}

type contractConfig struct {
	Enabled bool
	Labels  []string
	Nested  *contractNested
}

type contractNested struct {
	Name string
}

func (c *contractConfig) Clone() *contractConfig {
	if c == nil {
		return nil
	}
	out := *c
	out.Labels = slices.Clone(c.Labels)
	if c.Nested != nil {
		nested := *c.Nested
		out.Nested = &nested
	}
	return &out
}

func mutateContractConfig(c *contractConfig) {
	if c == nil {
		return
	}
	c.Enabled = !c.Enabled
	if len(c.Labels) > 0 {
		c.Labels[0] = "mutated"
	}
	c.Labels = append(c.Labels, "extra")
	if c.Nested != nil {
		c.Nested.Name = "mutated"
	}
}

func equalContractConfig(got, want *contractConfig) bool {
	if got == nil || want == nil {
		return got == want
	}
	if got.Enabled != want.Enabled || !slices.Equal(got.Labels, want.Labels) {
		return false
	}
	if got.Nested == nil || want.Nested == nil {
		return got.Nested == want.Nested
	}
	return got.Nested.Name == want.Nested.Name
}

type staticEnabled bool

func (e staticEnabled) IsEnabled() bool { return bool(e) }

type contractRuntime struct {
	Name string
}

type contractRuntimePlugin struct {
	contractPlugin
}

func (p *contractRuntimePlugin) Runtime(_ context.Context, _ *plugin.AppContext) (any, error) {
	return &contractRuntime{Name: p.Name()}, nil
}

type contractContributor struct{}

func (contractContributor) Name() string        { return "contract" }
func (contractContributor) Description() string { return "contract contributor" }

func (contractContributor) PipelineContribution(*plugin.AppContext) (*pipeline.Contribution, error) {
	first, err := pipeline.NewContributedJob(pipeline.ContributedJobOptions{Name: "first", Commands: []string{"first"}})
	if err != nil {
		return nil, err
	}
	second, err := pipeline.NewContributedJob(pipeline.ContributedJobOptions{Name: "second", Commands: []string{"second"}})
	if err != nil {
		return nil, err
	}
	contribution, err := pipeline.NewContribution(first, second)
	if err != nil {
		return nil, err
	}
	return contribution, nil
}

type contractPreflightPlugin struct{}

func (contractPreflightPlugin) Name() string        { return "contract-preflight" }
func (contractPreflightPlugin) Description() string { return "contract preflight" }
func (contractPreflightPlugin) Preflight(context.Context, *plugin.AppContext) error {
	return nil
}

type contractInitContributor struct{}

func (contractInitContributor) Name() string        { return "contract-init" }
func (contractInitContributor) Description() string { return "contract init" }

func (contractInitContributor) InitGroups() []*initwiz.InitGroupSpec {
	return []*initwiz.InitGroupSpec{{
		Title:    "Contract",
		Category: initwiz.CategoryFeature,
		Order:    10,
		Fields: []initwiz.InitField{
			initwiz.NewBoolField(initwiz.BoolFieldOptions{
				Key:   initwiz.MustStateKey[bool]("contract_enabled"),
				Title: "Enable contract",
			}),
		},
	}}
}

func (contractInitContributor) BuildInitConfig(*initwiz.StateMap) (*initwiz.InitContribution, error) {
	return initwiz.NewInitContribution("contract", contractInitConfig{Enabled: true})
}

type contractInitConfig struct {
	Enabled bool `yaml:"enabled"`
}

type contractVersionProvider struct{}

func (contractVersionProvider) Name() string        { return "contract-version" }
func (contractVersionProvider) Description() string { return "contract version" }

func (contractVersionProvider) VersionInfo() map[string]string {
	return map[string]string{"engine": "test"}
}

type contractKVCacheProvider struct{}

func (contractKVCacheProvider) Name() string        { return "contract-kv" }
func (contractKVCacheProvider) Description() string { return "contract kv" }

func (contractKVCacheProvider) NewKVCache(context.Context, *plugin.AppContext) (plugin.KVCache, error) {
	return &contractKVCache{values: map[string][]byte{}}, nil
}

type contractKVCache struct {
	values map[string][]byte
}

func (c *contractKVCache) Get(_ context.Context, namespace, key string) (value []byte, found bool, err error) {
	value, ok := c.values[namespace+"/"+key]
	return slices.Clone(value), ok, nil
}

func (c *contractKVCache) Set(_ context.Context, namespace, key string, value []byte, _ time.Duration) error {
	c.values[namespace+"/"+key] = slices.Clone(value)
	return nil
}

func (c *contractKVCache) Delete(_ context.Context, namespace, key string) error {
	delete(c.values, namespace+"/"+key)
	return nil
}

func (c *contractKVCache) DeleteNamespace(_ context.Context, namespace string) error {
	prefix := namespace + "/"
	for key := range c.values {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.values, key)
		}
	}
	return nil
}

type contractBlobStoreProvider struct{}

func (contractBlobStoreProvider) Name() string        { return "contract-blob" }
func (contractBlobStoreProvider) Description() string { return "contract blob" }

func (contractBlobStoreProvider) NewBlobStore(context.Context, *plugin.AppContext, plugin.BlobStoreOptions) (blobcache.Store, error) {
	return blobtest.NewMemoryStore("contract"), nil
}

type contractChangeDetector struct{}

func (contractChangeDetector) DetectChanges(context.Context, workflow.ChangeDetectionRequest) (*workflow.ChangeDetectionResult, error) {
	return &workflow.ChangeDetectionResult{Files: []string{"changed.tf"}}, nil
}

type contractCIProvider struct{}

func (contractCIProvider) Name() string        { return "contract-ci" }
func (contractCIProvider) Description() string { return "contract ci" }
func (contractCIProvider) DetectEnv() bool     { return true }
func (contractCIProvider) ProviderName() string {
	return "contract-ci"
}
func (contractCIProvider) PipelineID() string { return "pipeline" }
func (contractCIProvider) CommitSHA() string  { return "commit" }

func (contractCIProvider) NewGenerator(*plugin.AppContext, *pipeline.IR) (pipeline.Generator, error) {
	return contractGenerator{}, nil
}

func (contractCIProvider) NewCommentService(*plugin.AppContext) ci.CommentService {
	return contractCommentService{}
}

type contractGenerator struct{}

func (contractGenerator) Generate() (pipeline.GeneratedPipeline, error) { return nil, nil }
func (contractGenerator) DryRun() (*pipeline.DryRunResult, error) {
	return &pipeline.DryRunResult{}, nil
}

type contractCommentService struct{}

func (contractCommentService) IsEnabled() bool { return true }
func (contractCommentService) UpsertComment(context.Context, string) error {
	return nil
}
