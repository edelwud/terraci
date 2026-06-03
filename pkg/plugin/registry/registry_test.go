package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/workflow"
)

type testPlugin struct {
	name string
	desc string
}

func mustContribution(tb testing.TB, name string) *pipeline.Contribution {
	tb.Helper()
	job, err := pipeline.NewContributedJob(pipeline.ContributedJobOptions{
		Name:     name,
		Commands: []string{"terraci " + name},
	})
	if err != nil {
		tb.Fatalf("NewContributedJob() error = %v", err)
	}
	contribution, err := pipeline.NewContribution(job)
	if err != nil {
		tb.Fatalf("NewContribution() error = %v", err)
	}
	return contribution
}

func (p *testPlugin) Name() string        { return p.name }
func (p *testPlugin) Description() string { return p.desc }

type testCommandPlugin struct {
	testPlugin
}

func (p *testCommandPlugin) Commands() []*cobra.Command { return []*cobra.Command{{Use: p.name}} }

type testVersionPlugin struct {
	testPlugin
}

func (p *testVersionPlugin) VersionInfo() map[string]string {
	return map[string]string{"test": p.name}
}

type testCIInfoPlugin struct {
	testPlugin
}

func (p *testCIInfoPlugin) ProviderName() string { return p.name }
func (p *testCIInfoPlugin) PipelineID() string   { return "" }
func (p *testCIInfoPlugin) CommitSHA() string    { return "" }

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
	contribution    *pipeline.Contribution
	contributionErr error
	gateEnabled     *bool
	gateErr         error
	seenCtx         *plugin.AppContext
}

func (p *testContributorPlugin) PipelineContribution(ctx *plugin.AppContext) (*pipeline.Contribution, error) {
	p.seenCtx = ctx
	return p.contribution, p.contributionErr
}

func (p *testContributorPlugin) PipelineContributionEnabled(*plugin.AppContext) (bool, error) {
	if p.gateErr != nil {
		return false, p.gateErr
	}
	if p.gateEnabled == nil {
		return true, nil
	}
	return *p.gateEnabled, nil
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
	store blobcache.Store
}

func (p *testBlobStoreProvider) NewBlobStore(context.Context, *plugin.AppContext, plugin.BlobStoreOptions) (blobcache.Store, error) {
	return p.store, nil
}

type testBlobStore struct{}

func (testBlobStore) Get(context.Context, string, string) (data []byte, ok bool, meta blobcache.Meta, err error) {
	return nil, false, blobcache.Meta{}, nil
}
func (testBlobStore) Put(context.Context, string, string, []byte, blobcache.PutOptions) (blobcache.Meta, error) {
	return blobcache.Meta{}, nil
}
func (testBlobStore) Open(context.Context, string, string) (io.ReadCloser, bool, blobcache.Meta, error) {
	return nil, false, blobcache.Meta{}, nil
}
func (testBlobStore) PutStream(context.Context, string, string, io.Reader, blobcache.PutOptions) (blobcache.Meta, error) {
	return blobcache.Meta{}, nil
}
func (testBlobStore) Delete(context.Context, string, string) error             { return nil }
func (testBlobStore) DeleteNamespace(context.Context, string) error            { return nil }
func (testBlobStore) List(context.Context, string) ([]blobcache.Object, error) { return nil, nil }

type testConfig struct {
	Name    string
	Enabled bool
}

func (c *testConfig) Clone() *testConfig {
	if c == nil {
		return nil
	}
	out := *c
	return &out
}

func TestRegisterAndLookupCommandPlugin(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "test", desc: "A test plugin"} })

	got, ok := New().LookupCommandPlugin("test")
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

// invalidPlugin embeds BasePlugin in the misconfigured EnabledExplicitly +
// nil-IsEnabledFn shape that previously silently disabled the plugin at
// runtime. RegisterFactory must now reject it loudly.
type invalidPlugin struct {
	plugin.BasePlugin[*invalidConfig]
}

type invalidConfig struct{}

func (c *invalidConfig) Clone() *invalidConfig {
	if c == nil {
		return nil
	}
	out := *c
	return &out
}

func TestRegisterFactory_RejectsExplicitWithoutIsEnabledFn(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for EnabledExplicitly without IsEnabledFn")
		}
		msg, _ := r.(string)
		if msg == "" || !contains(msg, "EnabledExplicitly") {
			t.Fatalf("panic message = %v, want mention of EnabledExplicitly", r)
		}
	}()

	RegisterFactory(func() plugin.Plugin {
		return &invalidPlugin{
			BasePlugin: plugin.BasePlugin[*invalidConfig]{
				PluginName: "broken",
				PluginDesc: "broken plugin",
				EnableMode: plugin.EnabledExplicitly,
				DefaultCfg: func() *invalidConfig { return &invalidConfig{} },
				// IsEnabledFn intentionally nil
			},
		}
	})
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
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

	// The package-private generic helper still powers typed capability views.
	type hasName interface {
		plugin.Plugin
		Name() string
	}
	named := byCapabilityFrom[hasName](New())
	if len(named) != 2 {
		t.Fatalf("got %d named plugins, want 2", len(named))
	}
}

func TestLifecycleFacadesPreserveOrder(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	var preflightCalled string
	RegisterFactory(func() plugin.Plugin {
		return &testPreflightPlugin{
			BasePlugin: plugin.BasePlugin[*testConfig]{
				PluginName: "config",
				PluginDesc: "config plugin",
				EnableMode: plugin.EnabledByDefault,
				DefaultCfg: func() *testConfig { return &testConfig{} },
			},
			called: &preflightCalled,
		}
	})
	RegisterFactory(func() plugin.Plugin { return &testCommandPlugin{testPlugin: testPlugin{name: "command"}} })
	RegisterFactory(func() plugin.Plugin { return &testVersionPlugin{testPlugin: testPlugin{name: "version"}} })
	RegisterFactory(func() plugin.Plugin { return &testCIInfoPlugin{testPlugin: testPlugin{name: "ci"}} })

	plugins := New()
	if schemas := plugins.ExtensionSchemas(); schemas["config"] == nil {
		t.Fatalf("ExtensionSchemas() missing config schema: %#v", schemas)
	}
	commands := plugins.Commands()
	if len(commands) != 1 || commands[0].Use != "command" {
		t.Fatalf("Commands() = %#v, want command", commands)
	}
	version := plugins.VersionSnapshot()
	if got := version.Info()["test"]; got != "version" {
		t.Fatalf("VersionSnapshot().Info()[test] = %q, want version", got)
	}
	initSnapshot, err := plugins.InitWizardSnapshot()
	if err != nil {
		t.Fatalf("InitWizardSnapshot() error = %v", err)
	}
	providers := initSnapshot.ProviderOptions()
	if len(providers) != 1 || providers[0].Name() != "ci" {
		t.Fatalf("InitWizardSnapshot().ProviderOptions() = %#v, want ci", providers)
	}
	if err := plugins.RunPreflight(context.Background(), nil); err != nil {
		t.Fatalf("RunPreflight() error = %v", err)
	}
	if preflightCalled != "preflight" {
		t.Fatalf("preflight called = %q, want preflight", preflightCalled)
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

	firstPlugin := byCapabilityFrom[plugin.ConfigLoader](first)[0]
	secondPlugin := byCapabilityFrom[plugin.ConfigLoader](second)[0]
	if firstPlugin == secondPlugin {
		t.Fatal("New() returned shared plugin instances")
	}

	if err := firstPlugin.DecodeAndSet(testExtensionDocument(t, "isolated", &testConfig{Enabled: true})); err != nil {
		t.Fatalf("DecodeAndSet() error = %v", err)
	}

	if !firstPlugin.IsEnabled() {
		t.Fatal("first plugin should be enabled after decode")
	}
	if secondPlugin.IsEnabled() {
		t.Fatal("second plugin observed config state from first plugin")
	}
}

func testExtensionDocument(tb testing.TB, key string, value any) config.ExtensionDocument {
	tb.Helper()
	extensionValue, err := config.NewExtensionValue(key, value)
	if err != nil {
		tb.Fatalf("NewExtensionValue() error = %v", err)
	}
	set, err := config.NewExtensionSet(extensionValue)
	if err != nil {
		tb.Fatalf("NewExtensionSet() error = %v", err)
	}
	cfg, err := config.Build(config.BuildOptions{Extensions: set})
	if err != nil {
		tb.Fatalf("config.Build() error = %v", err)
	}
	doc, ok := cfg.Extension(config.MustExtensionKey(key))
	if !ok {
		tb.Fatalf("Extension(%q) missing", key)
	}
	return doc
}

func TestCatalogCreatesIndependentPluginSets(t *testing.T) {
	catalog := NewCatalog()
	catalog.RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "first"} })

	other := NewCatalog()
	other.RegisterFactory(func() plugin.Plugin { return &testPlugin{name: "second"} })

	if _, ok := catalog.NewRegistry().LookupCommandPlugin("second"); ok {
		t.Fatal("catalog observed plugin from another catalog")
	}
	if _, ok := other.NewRegistry().LookupCommandPlugin("first"); ok {
		t.Fatal("other catalog observed plugin from catalog")
	}
}

func TestLookupCommandPluginNotFound(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	_, ok := New().LookupCommandPlugin("nonexistent")
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

func TestResolveKVCacheProvider_AmbiguousUsesConfigPathHint(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	RegisterFactory(func() plugin.Plugin {
		return &testKVCacheProvider{
			testPlugin: testPlugin{name: "cache-a", desc: "cache backend A"},
			cache:      testKVCache{},
		}
	})
	RegisterFactory(func() plugin.Plugin {
		return &testKVCacheProvider{
			testPlugin: testPlugin{name: "cache-b", desc: "cache backend B"},
			cache:      testKVCache{},
		}
	})

	_, err := New().ResolveKVCacheProvider("", "set extensions.feature.cache.backend explicitly")
	if err == nil {
		t.Fatal("expected ambiguous backend error")
	}
	if !strings.Contains(err.Error(), "set extensions.feature.cache.backend explicitly") {
		t.Fatalf("ResolveKVCacheProvider() error = %v, want config path hint", err)
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

	contribs, err := New().CollectContributions(nil)
	if err != nil {
		t.Fatalf("CollectContributions() error = %v", err)
	}
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

	// Version provider capability is not implemented by testPlugin.
	vp := byCapabilityFrom[plugin.VersionProvider](New())
	if len(vp) != 0 {
		t.Errorf("expected 0 version providers, got %d", len(vp))
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
		contribution: mustContribution(t, "enabled-job"),
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
		contribution: mustContribution(t, "disabled-job"),
	}
	disabled.SetTypedConfig(&testConfig{Enabled: false})
	plugins := NewFromFactories(
		func() plugin.Plugin { return enabled },
		func() plugin.Plugin { return disabled },
	)

	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		Config:     config.DefaultConfig(),
		WorkDir:    "/work",
		ServiceDir: "/service",
		Version:    "test",
		Reports:    ci.NewFileReportStore("/service"),
	})
	contribs, err := plugins.CollectContributions(appCtx)
	if err != nil {
		t.Fatalf("CollectContributions() error = %v", err)
	}
	if len(contribs) != 1 {
		t.Fatalf("expected 1 contribution, got %d", len(contribs))
	}
	jobs := contribs[0].Jobs()
	if len(jobs) != 1 || jobs[0].Name() != "enabled-job" {
		t.Errorf("expected enabled-job, got %#v", jobs)
	}
	if enabled.seenCtx != appCtx {
		t.Fatal("enabled contributor did not receive app context")
	}
}

func TestCollectContributions_GateFalseSkipsContributor(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	gateEnabled := false
	contributor := newEnabledContributor(t, "gated", mustContribution(t, "gated-job"))
	contributor.gateEnabled = &gateEnabled

	contribs, err := NewFromFactories(func() plugin.Plugin { return contributor }).CollectContributions(nil)
	if err != nil {
		t.Fatalf("CollectContributions() error = %v", err)
	}
	if len(contribs) != 0 {
		t.Fatalf("CollectContributions() returned %d contributions, want 0", len(contribs))
	}
	if contributor.seenCtx != nil {
		t.Fatal("PipelineContribution() was called even though gate returned false")
	}
}

func TestCollectContributions_GateErrorIsTyped(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	gateErr := errors.New("gate failed")
	contributor := newEnabledContributor(t, "broken-gate", mustContribution(t, "job"))
	contributor.gateErr = gateErr

	_, err := NewFromFactories(func() plugin.Plugin { return contributor }).CollectContributions(nil)
	assertPipelineContributionError(t, err, "broken-gate", plugin.PipelineContributionPhaseGate, gateErr)
}

func TestCollectContributions_ContributorErrorIsTyped(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	contributionErr := errors.New("job build failed")
	contributor := newEnabledContributor(t, "broken-contribution", nil)
	contributor.contributionErr = contributionErr

	_, err := NewFromFactories(func() plugin.Plugin { return contributor }).CollectContributions(nil)
	assertPipelineContributionError(t, err, "broken-contribution", plugin.PipelineContributionPhaseContribution, contributionErr)
}

func TestCollectContributions_NilContributionIsError(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	contributor := newEnabledContributor(t, "nil-contribution", nil)

	_, err := NewFromFactories(func() plugin.Plugin { return contributor }).CollectContributions(nil)
	assertPipelineContributionError(
		t,
		err,
		"nil-contribution",
		plugin.PipelineContributionPhaseContribution,
		plugin.ErrNilPipelineContribution,
	)
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
		contribution: mustContribution(t, "bare-job"),
	}

	// bareContributor doesn't satisfy PipelineContributor since it doesn't have PipelineContribution()
	// So CollectContributions won't find it — this is expected behavior
	contribs, err := NewFromFactories(func() plugin.Plugin { return p }).CollectContributions(nil)
	if err != nil {
		t.Fatalf("CollectContributions() error = %v", err)
	}
	if len(contribs) != 0 {
		t.Errorf("expected 0 contributions from bare plugin, got %d", len(contribs))
	}
}

func newEnabledContributor(tb testing.TB, name string, contribution *pipeline.Contribution) *testContributorPlugin {
	tb.Helper()
	contributor := &testContributorPlugin{
		BasePlugin: plugin.BasePlugin[*testConfig]{
			PluginName: name,
			PluginDesc: name + " plugin",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
		contribution: contribution,
	}
	contributor.SetTypedConfig(&testConfig{Enabled: true})
	return contributor
}

func assertPipelineContributionError(
	tb testing.TB,
	err error,
	wantPlugin string,
	wantPhase plugin.PipelineContributionPhase,
	wantWrapped error,
) {
	tb.Helper()
	if err == nil {
		tb.Fatal("CollectContributions() error = nil")
	}
	var contributionErr *plugin.PipelineContributionError
	if !errors.As(err, &contributionErr) {
		tb.Fatalf("CollectContributions() error type = %T, want *plugin.PipelineContributionError", err)
	}
	if contributionErr.Plugin != wantPlugin {
		tb.Fatalf("PipelineContributionError.Plugin = %q, want %q", contributionErr.Plugin, wantPlugin)
	}
	if contributionErr.Phase != wantPhase {
		tb.Fatalf("PipelineContributionError.Phase = %q, want %q", contributionErr.Phase, wantPhase)
	}
	if !errors.Is(err, wantWrapped) {
		tb.Fatalf("CollectContributions() error = %v, want wrapping %v", err, wantWrapped)
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

	first := byCapabilityFrom[plugin.ConfigLoader](New())[0]
	first.(*testContributorPlugin).SetTypedConfig(&testConfig{Name: "configured"})
	if !first.IsConfigured() {
		t.Fatal("first plugin should be configured")
	}

	second := byCapabilityFrom[plugin.ConfigLoader](New())[0]
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
func (p *testProviderPlugin) NewGenerator(_ *pipeline.IR) (pipeline.Generator, error) {
	return nil, nil
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

func TestRunPreflight_FiltersDisabledPlugins(t *testing.T) {
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

	var called string
	plugins := NewFromFactories(
		func() plugin.Plugin { return disabled },
		func() plugin.Plugin { return enabled },
	)
	enabled.called = &called
	if err := plugins.RunPreflight(context.Background(), nil); err != nil {
		t.Fatalf("RunPreflight() error = %v", err)
	}
	if called != "preflight" {
		t.Fatalf("enabled preflight called = %q, want preflight", called)
	}
}

// --- ChangeDetectionProvider tests ---

type testDetectorPlugin struct {
	plugin.BasePlugin[*testConfig]
}

func (d *testDetectorPlugin) DetectChanges(_ context.Context, _ workflow.ChangeDetectionRequest) (*workflow.ChangeDetectionResult, error) {
	return &workflow.ChangeDetectionResult{}, nil
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
func (d *bareDetector) DetectChanges(_ context.Context, _ workflow.ChangeDetectionRequest) (*workflow.ChangeDetectionResult, error) {
	return &workflow.ChangeDetectionResult{}, nil
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
			_, _ = New().LookupCommandPlugin("plugin-0")
			_ = byCapabilityFrom[plugin.Plugin](New())
		})
	}
	wg.Wait()

	all := New().All()
	if len(all) != 10 {
		t.Fatalf("expected 10 plugins after concurrent access, got %d", len(all))
	}
}
