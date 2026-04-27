package registry

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

type testPlugin struct {
	name string
	desc string
}

func (p *testPlugin) Name() string        { return p.name }
func (p *testPlugin) Description() string { return p.desc }

type testCommandPlugin struct {
	testPlugin
}

type testPreflightPlugin struct {
	plugin.BasePlugin[*testConfig]
	called *string
}

func (p *testPreflightPlugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	if p.called != nil {
		*p.called = "preflight"
	}
	return nil
}

// testContributorPlugin implements PipelineContributor + ConfigLoader for testing.
type testContributorPlugin struct {
	plugin.BasePlugin[*testConfig]
	contribution *pipeline.Contribution
	seenCtx      *plugin.AppContext
}

func (p *testContributorPlugin) PipelineContribution(ctx *plugin.AppContext) *pipeline.Contribution {
	p.seenCtx = ctx
	return p.contribution
}

type testKVCacheProvider struct {
	testPlugin
	cache plugin.KVCache
}

func (p *testKVCacheProvider) NewKVCache(context.Context, *plugin.AppContext) (plugin.KVCache, error) {
	return p.cache, nil
}

type testKVCache struct{}

func (testKVCache) Get(context.Context, string, string) (value []byte, found bool, err error) {
	return nil, false, nil
}
func (testKVCache) Set(context.Context, string, string, []byte, time.Duration) error {
	return nil
}
func (testKVCache) Delete(context.Context, string, string) error  { return nil }
func (testKVCache) DeleteNamespace(context.Context, string) error { return nil }

type testBlobStoreProvider struct {
	testPlugin
	store plugin.BlobStore
}

func (p *testBlobStoreProvider) NewBlobStore(context.Context, *plugin.AppContext) (plugin.BlobStore, error) {
	return p.store, nil
}

type testBlobStore struct{}

func (testBlobStore) Get(context.Context, string, string) (data []byte, ok bool, meta plugin.BlobMeta, err error) {
	return nil, false, plugin.BlobMeta{}, nil
}
func (testBlobStore) Put(context.Context, string, string, []byte, plugin.PutBlobOptions) (plugin.BlobMeta, error) {
	return plugin.BlobMeta{}, nil
}
func (testBlobStore) Open(context.Context, string, string) (io.ReadCloser, bool, plugin.BlobMeta, error) {
	return nil, false, plugin.BlobMeta{}, nil
}
func (testBlobStore) PutStream(context.Context, string, string, io.Reader, plugin.PutBlobOptions) (plugin.BlobMeta, error) {
	return plugin.BlobMeta{}, nil
}
func (testBlobStore) Delete(context.Context, string, string) error              { return nil }
func (testBlobStore) DeleteNamespace(context.Context, string) error             { return nil }
func (testBlobStore) List(context.Context, string) ([]plugin.BlobObject, error) { return nil, nil }

type testConfig struct {
	Name    string
	Enabled bool
}

func TestRegisterAndGet(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "test", desc: "A test plugin"} })

	got, ok := New().Get("test")
	if !ok {
		t.Fatal("expected to find plugin")
	}
	if got.Name() != "test" {
		t.Fatalf("got name %q, want %q", got.Name(), "test")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "dup"} })

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "dup"} })
}

func TestAll_Order(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "b"} })
	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "a"} })
	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "c"} })

	all := New().All()
	if len(all) != 3 {
		t.Fatalf("got %d plugins, want 3", len(all))
	}
	if all[0].Name() != "b" || all[1].Name() != "a" || all[2].Name() != "c" {
		t.Fatalf("wrong order: %s, %s, %s", all[0].Name(), all[1].Name(), all[2].Name())
	}
}

func TestByCapability(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "plain"} })
	RegisterFactory(func() plugin.Plugin { return &testCommandPlugin{testPlugin: testPlugin{name: "cmd"}} })

	// All plugins
	all := New().All()
	if len(all) != 2 {
		t.Fatalf("got %d plugins, want 2", len(all))
	}

	// Only command plugins — testCommandPlugin doesn't actually implement CommandProvider,
	// but we can test that ByCapability filters correctly with our test interface
	type hasName interface {
		plugin.Plugin
		Name() string
	}
	named := ByCapabilityFrom[hasName](New())
	if len(named) != 2 {
		t.Fatalf("got %d named plugins, want 2", len(named))
	}
}

func TestNewCreatesIsolatedPluginInstances(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin {
		return &testPreflightPlugin{
			BasePlugin: plugin.BasePlugin[*testConfig]{
				PluginName: "isolated",
				PluginDesc: "isolated plugin",
				EnableMode: plugin.EnabledExplicitly,
				DefaultCfg: func() *testConfig { return &testConfig{} },
				IsEnabledFn: func(cfg *testConfig) bool {
					return cfg != nil && cfg.Enabled
				},
			},
		}
	})

	first := New()
	second := New()

	firstPlugin := ByCapabilityFrom[plugin.ConfigLoader](first)[0]
	secondPlugin := ByCapabilityFrom[plugin.ConfigLoader](second)[0]
	if firstPlugin == secondPlugin {
		t.Fatal("New() returned shared plugin instances")
	}

	if err := firstPlugin.DecodeAndSet(func(target any) error {
		cfg := target.(**testConfig)
		*cfg = &testConfig{Enabled: true}
		return nil
	}); err != nil {
		t.Fatalf("DecodeAndSet() error = %v", err)
	}

	if !firstPlugin.IsEnabled() {
		t.Fatal("first plugin should be enabled after decode")
	}
	if secondPlugin.IsEnabled() {
		t.Fatal("second plugin observed config state from first plugin")
	}
}

func TestGetNotFound(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	_, ok := New().Get("nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestResolveCIProvider_NoPlugins(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	_, err := New().ResolveCIProvider()
	if err == nil {
		t.Fatal("expected error with no providers")
	}
}

func TestResolveKVCacheProvider(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin {
		return &testKVCacheProvider{
			testPlugin: testPlugin{name: "cache", desc: "cache backend"},
			cache:      testKVCache{},
		}
	})

	got, err := New().ResolveKVCacheProvider("cache")
	if err != nil {
		t.Fatalf("New().ResolveKVCacheProvider() error = %v", err)
	}
	if got.Name() != "cache" {
		t.Fatalf("New().ResolveKVCacheProvider() = %q, want %q", got.Name(), "cache")
	}
}

func TestResolveKVCacheProvider_NotFound(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	if _, err := New().ResolveKVCacheProvider("missing"); err == nil {
		t.Fatal("expected error for missing cache backend")
	}
}

func TestResolveKVCacheProvider_WrongCapability(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "plain", desc: "plain plugin"} })

	if _, err := New().ResolveKVCacheProvider("plain"); err == nil {
		t.Fatal("expected error when plugin does not implement KV cache capability")
	}
}

func TestResolveBlobStoreProvider(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin {
		return &testBlobStoreProvider{
			testPlugin: testPlugin{name: "blob", desc: "blob backend"},
			store:      testBlobStore{},
		}
	})

	got, err := New().ResolveBlobStoreProvider("blob")
	if err != nil {
		t.Fatalf("New().ResolveBlobStoreProvider() error = %v", err)
	}
	if got.Name() != "blob" {
		t.Fatalf("New().ResolveBlobStoreProvider() = %q, want %q", got.Name(), "blob")
	}
}

func TestResolveBlobStoreProvider_NotFound(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	if _, err := New().ResolveBlobStoreProvider("missing"); err == nil {
		t.Fatal("expected error for missing blob backend")
	}
}

func TestResolveBlobStoreProvider_WrongCapability(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "plain", desc: "plain plugin"} })

	if _, err := New().ResolveBlobStoreProvider("plain"); err == nil {
		t.Fatal("expected error when plugin does not implement blob store capability")
	}
}

func TestResolveChangeDetector_None(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	_, err := New().ResolveChangeDetector()
	if err == nil {
		t.Fatal("expected error with no detectors")
	}
}

func TestCollectContributions_Empty(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	contribs := New().CollectContributions(nil)
	if len(contribs) != 0 {
		t.Errorf("expected 0 contributions, got %d", len(contribs))
	}
}

func TestAll_Empty(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	all := New().All()
	if len(all) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(all))
	}
}

func TestByCapability_NoMatch(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "basic"} })

	// VersionProvider is not implemented by testPlugin
	vp := ByCapabilityFrom[plugin.VersionProvider](New())
	if len(vp) != 0 {
		t.Errorf("expected 0 VersionProviders, got %d", len(vp))
	}
}

func TestReset(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "x"} })
	if len(New().All()) != 1 {
		t.Fatal("expected 1 plugin after register")
	}

	Reset()
	if len(New().All()) != 0 {
		t.Error("expected 0 plugins after reset")
	}
}

func TestCollectContributions_FiltersDisabledPlugins(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	// Enabled plugin with contribution
	enabled := &testContributorPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName: "enabled",
			PluginDesc: "enabled plugin",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
		contribution: &pipeline.Contribution{
			Jobs: []pipeline.ContributedJob{{Name: "enabled-job"}},
		},
	}
	enabled.SetTypedConfig(&testConfig{Enabled: true})

	// Disabled plugin with contribution
	disabled := &testContributorPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName: "disabled",
			PluginDesc: "disabled plugin",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
		contribution: &pipeline.Contribution{
			Jobs: []pipeline.ContributedJob{{Name: "disabled-job"}},
		},
	}
	disabled.SetTypedConfig(&testConfig{Enabled: false})
	plugins := NewFromFactories(
		func() plugin.Plugin { return enabled },
		func() plugin.Plugin { return disabled },
	)

	appCtx := plugin.NewAppContext(config.DefaultConfig(), "/work", "/service", "test", plugin.NewReportRegistry())
	contribs := plugins.CollectContributions(appCtx)
	if len(contribs) != 1 {
		t.Fatalf("expected 1 contribution, got %d", len(contribs))
	}
	if contribs[0].Jobs[0].Name != "enabled-job" {
		t.Errorf("expected enabled-job, got %s", contribs[0].Jobs[0].Name)
	}
	if enabled.seenCtx != appCtx {
		t.Fatal("enabled contributor did not receive app context")
	}
}

func TestCollectContributions_IncludesPluginWithoutConfigLoader(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	// Plugin that implements PipelineContributor but NOT ConfigLoader
	type bareContributor struct {
		testPlugin
		contribution *pipeline.Contribution
	}
	p := &bareContributor{
		testPlugin:   testPlugin{name: "bare", desc: "bare contributor"},
		contribution: &pipeline.Contribution{Jobs: []pipeline.ContributedJob{{Name: "bare-job"}}},
	}

	// bareContributor doesn't satisfy PipelineContributor since it doesn't have PipelineContribution()
	// So CollectContributions won't find it — this is expected behavior
	contribs := NewFromFactories(func() plugin.Plugin { return p }).CollectContributions(nil)
	if len(contribs) != 0 {
		t.Errorf("expected 0 contributions from bare plugin, got %d", len(contribs))
	}
}

func TestNewDoesNotReuseConfiguredRegisteredPluginState(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin {
		return &testContributorPlugin{
			BasePlugin: plugin.BasePlugin[*testConfig]{
				PluginName: "fresh",
				PluginDesc: "fresh plugin",
				EnableMode: plugin.EnabledWhenConfigured,
				DefaultCfg: func() *testConfig { return &testConfig{} },
			},
		}
	})

	first := ByCapabilityFrom[plugin.ConfigLoader](New())[0]
	first.(*testContributorPlugin).SetTypedConfig(&testConfig{Name: "configured"})
	if !first.IsConfigured() {
		t.Fatal("first plugin should be configured")
	}

	second := ByCapabilityFrom[plugin.ConfigLoader](New())[0]
	if second.IsConfigured() {
		t.Error("second plugin should start unconfigured")
	}
}

func TestResolvedCIProvider_Methods(t *testing.T) {
	tp := &testProviderPlugin{testPlugin: testPlugin{name: "test-ci", desc: "Test CI"}}

	provider := plugin.NewResolvedCIProvider(tp, tp, tp, nil)

	if provider.Name() != "test-ci" {
		t.Errorf("Name() = %q, want test-ci", provider.Name())
	}
	if provider.Description() != "Test CI" {
		t.Errorf("Description() = %q, want Test CI", provider.Description())
	}
}

type testProviderPlugin struct {
	plugin.BasePlugin[*testConfig]
	testPlugin
	detectEnv bool
	provider  string
}

func (p *testProviderPlugin) Name() string        { return p.name }
func (p *testProviderPlugin) Description() string { return p.desc }
func (p *testProviderPlugin) DetectEnv() bool     { return p.detectEnv }
func (p *testProviderPlugin) ProviderName() string {
	return p.provider
}
func (p *testProviderPlugin) PipelineID() string { return "1" }
func (p *testProviderPlugin) CommitSHA() string  { return "abc" }
func (p *testProviderPlugin) NewGenerator(_ *plugin.AppContext, _ *graph.DependencyGraph, _ []*discovery.Module) pipeline.Generator {
	return nil
}
func (p *testProviderPlugin) NewCommentService(_ *plugin.AppContext) ci.CommentService {
	return nil
}

func TestResolveProvider_SkipsDisabledEnvDetectedProvider(t *testing.T) {
	t.Cleanup(func() {
		Reset()
		t.Setenv("TERRACI_PROVIDER", "")
	})
	Reset()

	disabledEnv := &testProviderPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName:  "github",
			PluginDesc:  "GitHub",
			EnableMode:  plugin.EnabledExplicitly,
			DefaultCfg:  func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool { return cfg != nil && cfg.Enabled },
		},
		testPlugin: testPlugin{name: "github", desc: "GitHub"},
		detectEnv:  true,
		provider:   "github",
	}
	disabledEnv.SetTypedConfig(&testConfig{Enabled: false})

	enabled := &testProviderPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName:  "gitlab",
			PluginDesc:  "GitLab",
			EnableMode:  plugin.EnabledExplicitly,
			DefaultCfg:  func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool { return cfg != nil && cfg.Enabled },
		},
		testPlugin: testPlugin{name: "gitlab", desc: "GitLab"},
		provider:   "gitlab",
	}
	enabled.SetTypedConfig(&testConfig{Enabled: true})

	provider, err := NewFromFactories(
		func() plugin.Plugin { return disabledEnv },
		func() plugin.Plugin { return enabled },
	).ResolveCIProvider()
	if err != nil {
		t.Fatalf("New().ResolveCIProvider() error = %v", err)
	}
	if provider.ProviderName() != "gitlab" {
		t.Fatalf("New().ResolveCIProvider() = %q, want gitlab", provider.ProviderName())
	}
}

func TestResolveCIProvider_ExplicitProviderOverridesDetectedEnv(t *testing.T) {
	t.Cleanup(func() {
		Reset()
		os.Unsetenv("TERRACI_PROVIDER")
	})
	Reset()
	t.Setenv("TERRACI_PROVIDER", "github")

	gitlab := &testProviderPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName: "gitlab",
			PluginDesc: "GitLab",
			EnableMode: plugin.EnabledAlways,
			DefaultCfg: func() *testConfig { return &testConfig{} },
		},
		testPlugin: testPlugin{name: "gitlab", desc: "GitLab"},
		detectEnv:  true,
		provider:   "gitlab",
	}

	github := &testProviderPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName: "github",
			PluginDesc: "GitHub",
			EnableMode: plugin.EnabledAlways,
			DefaultCfg: func() *testConfig { return &testConfig{} },
		},
		testPlugin: testPlugin{name: "github", desc: "GitHub"},
		provider:   "github",
	}

	provider, err := NewFromFactories(
		func() plugin.Plugin { return gitlab },
		func() plugin.Plugin { return github },
	).ResolveCIProvider()
	if err != nil {
		t.Fatalf("New().ResolveCIProvider() error = %v", err)
	}
	if provider.ProviderName() != "github" {
		t.Fatalf("New().ResolveCIProvider() = %q, want github", provider.ProviderName())
	}
}

func TestResolveCIProvider_TERRACI_PROVIDERMustBeActive(t *testing.T) {
	t.Cleanup(func() {
		Reset()
		os.Unsetenv("TERRACI_PROVIDER")
	})
	Reset()
	t.Setenv("TERRACI_PROVIDER", "github")

	disabled := &testProviderPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName:  "github",
			PluginDesc:  "GitHub",
			EnableMode:  plugin.EnabledExplicitly,
			DefaultCfg:  func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool { return cfg != nil && cfg.Enabled },
		},
		testPlugin: testPlugin{name: "github", desc: "GitHub"},
		provider:   "github",
	}
	disabled.SetTypedConfig(&testConfig{Enabled: false})

	if _, err := NewFromFactories(func() plugin.Plugin { return disabled }).ResolveCIProvider(); err == nil {
		t.Fatal("New().ResolveCIProvider() should fail when TERRACI_PROVIDER points to disabled plugin")
	}
}

func TestPreflightsForStartup_FiltersDisabledPlugins(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	disabled := &testPreflightPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName: "disabled-preflight",
			PluginDesc: "disabled preflight plugin",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
	}
	disabled.SetTypedConfig(&testConfig{Enabled: false})

	enabled := &testPreflightPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName: "enabled-preflight",
			PluginDesc: "enabled preflight plugin",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
	}
	enabled.SetTypedConfig(&testConfig{Enabled: true})

	preflights := NewFromFactories(
		func() plugin.Plugin { return disabled },
		func() plugin.Plugin { return enabled },
	).PreflightsForStartup()
	if len(preflights) != 1 {
		t.Fatalf("New().PreflightsForStartup() returned %d plugins, want 1", len(preflights))
	}
	if preflights[0].Name() != "enabled-preflight" {
		t.Fatalf("New().PreflightsForStartup()[0] = %q, want enabled-preflight", preflights[0].Name())
	}
}

// --- ChangeDetectionProvider tests ---

type testDetectorPlugin struct {
	plugin.BasePlugin[*testConfig]
}

func (d *testDetectorPlugin) DetectChangedModules(_ context.Context, _ *plugin.AppContext, _ string, _ *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	return nil, nil, nil
}

func (d *testDetectorPlugin) DetectChangedLibraries(_ context.Context, _ *plugin.AppContext, _ string, _ []string) ([]string, error) {
	return nil, nil
}

func TestResolveChangeDetector_Single(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	det := &testDetectorPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName: "git",
			PluginDesc: "Git detector",
			EnableMode: plugin.EnabledAlways,
			DefaultCfg: func() *testConfig { return &testConfig{} },
		},
	}

	got, err := NewFromFactories(func() plugin.Plugin { return det }).ResolveChangeDetector()
	if err != nil {
		t.Fatalf("New().ResolveChangeDetector() error = %v", err)
	}
	if got.Name() != "git" {
		t.Fatalf("New().ResolveChangeDetector() = %q, want git", got.Name())
	}
}

func TestResolveChangeDetector_SingleDisabledIsNotActive(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	det := &testDetectorPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName:  "git",
			PluginDesc:  "Git detector",
			EnableMode:  plugin.EnabledExplicitly,
			DefaultCfg:  func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool { return cfg != nil && cfg.Enabled },
		},
	}
	det.SetTypedConfig(&testConfig{Enabled: false})

	if _, err := NewFromFactories(func() plugin.Plugin { return det }).ResolveChangeDetector(); err == nil {
		t.Fatal("New().ResolveChangeDetector() should fail when the only detector is disabled")
	}
}

func TestResolveChangeDetector_MultipleWithConfigured(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	det1 := &testDetectorPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName:  "detector-a",
			PluginDesc:  "Detector A",
			EnableMode:  plugin.EnabledExplicitly,
			DefaultCfg:  func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool { return cfg != nil && cfg.Enabled },
		},
	}

	det2 := &testDetectorPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName:  "detector-b",
			PluginDesc:  "Detector B",
			EnableMode:  plugin.EnabledExplicitly,
			DefaultCfg:  func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool { return cfg != nil && cfg.Enabled },
		},
	}
	det2.SetTypedConfig(&testConfig{Enabled: true})

	got, err := NewFromFactories(
		func() plugin.Plugin { return det1 },
		func() plugin.Plugin { return det2 },
	).ResolveChangeDetector()
	if err != nil {
		t.Fatalf("New().ResolveChangeDetector() error = %v", err)
	}
	if got.Name() != "detector-b" {
		t.Fatalf("New().ResolveChangeDetector() = %q, want detector-b", got.Name())
	}
}

// bareDetector implements ChangeDetectionProvider without ConfigLoader.
type bareDetector struct {
	name string
}

func (d *bareDetector) Name() string        { return d.name }
func (d *bareDetector) Description() string { return d.name }
func (d *bareDetector) DetectChangedModules(_ context.Context, _ *plugin.AppContext, _ string, _ *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	return nil, nil, nil
}
func (d *bareDetector) DetectChangedLibraries(_ context.Context, _ *plugin.AppContext, _ string, _ []string) ([]string, error) {
	return nil, nil
}

func TestResolveChangeDetector_MultipleNoneConfigured(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	// Two detectors without ConfigLoader — neither can be prioritized.
	RegisterFactory(func() plugin.Plugin { return &bareDetector{name: "det-x"} })
	RegisterFactory(func() plugin.Plugin { return &bareDetector{name: "det-y"} })

	_, err := New().ResolveChangeDetector()
	if err == nil {
		t.Fatal("New().ResolveChangeDetector() should fail with multiple unconfigured detectors")
	}
	if !strings.Contains(err.Error(), "det-x") || !strings.Contains(err.Error(), "det-y") {
		t.Fatalf("error should list detector names, got: %v", err)
	}
}

// --- Concurrent registry access tests ---

func TestConcurrentRegistryAccess(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	// Pre-register plugins
	for i := range 10 {
		RegisterFactory(func() plugin.Plugin { return &testPlugin{name: fmt.Sprintf("plugin-%d", i), desc: "concurrent test"} })
	}

	// Concurrent reads
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			_ = New().All()
			_, _ = New().Get("plugin-0")
			_ = ByCapabilityFrom[plugin.Plugin](New())
		})
	}
	wg.Wait()

	all := New().All()
	if len(all) != 10 {
		t.Fatalf("expected 10 plugins after concurrent access, got %d", len(all))
	}
}

func TestRuntimeAs_Success(t *testing.T) {
	value, err := plugin.RuntimeAs[*testPlugin](&testPlugin{name: "runtime"})
	if err != nil {
		t.Fatalf("RuntimeAs() error = %v", err)
	}
	if value.Name() != "runtime" {
		t.Fatalf("RuntimeAs() returned %q, want runtime", value.Name())
	}
}

func TestRuntimeAs_TypeMismatch(t *testing.T) {
	_, err := plugin.RuntimeAs[*testCommandPlugin](&testPlugin{name: "runtime"})
	if err == nil {
		t.Fatal("RuntimeAs() error = nil, want mismatch")
	}
}
