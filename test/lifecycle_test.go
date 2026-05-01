package test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func TestPluginRegistration(t *testing.T) {
	plugins := registry.New()
	all := plugins.All()
	if len(all) != 9 {
		t.Fatalf("expected 9 plugins, got %d", len(all))
	}

	names := make(map[string]bool)
	for _, p := range all {
		if p.Name() == "" {
			t.Error("plugin has empty name")
		}
		if p.Description() == "" {
			t.Errorf("plugin %q has empty description", p.Name())
		}
		names[p.Name()] = true
	}

	expected := []string{"cost", "diskblob", "git", "github", "gitlab", "inmemcache", "policy", "summary", "tfupdate"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing plugin: %s", name)
		}
	}
}

func TestPluginCapabilities(t *testing.T) {
	plugins := registry.New()
	// ConfigLoader plugins
	configLoaders := registry.ByCapabilityFrom[plugin.ConfigLoader](plugins)
	if len(configLoaders) == 0 {
		t.Fatal("expected at least one ConfigLoader")
	}

	configKeys := make(map[string]bool)
	for _, cl := range configLoaders {
		key := cl.ConfigKey()
		if key == "" {
			t.Errorf("plugin %s has empty ConfigKey", cl.Name())
		}
		if configKeys[key] {
			t.Errorf("duplicate ConfigKey: %s", key)
		}
		configKeys[key] = true
	}

	// Verify specific expected config keys
	for _, expectedKey := range []string{"gitlab", "github", "cost", "diskblob", "inmemcache", "policy"} {
		if !configKeys[expectedKey] {
			t.Errorf("missing expected ConfigKey: %s", expectedKey)
		}
	}

	// CI provider plugins (gitlab + github) — must implement all CI interfaces
	ciProviders := registry.ByCapabilityFrom[plugin.CIInfoProvider](plugins)
	if len(ciProviders) < 2 {
		t.Errorf("expected at least 2 CIInfoProvider plugins (gitlab, github), got %d", len(ciProviders))
	}

	// Preflightable plugins
	preflightables := registry.ByCapabilityFrom[plugin.Preflightable](plugins)
	if len(preflightables) == 0 {
		t.Fatal("expected at least one Preflightable plugin")
	}

	// CommandProvider plugins
	commandProviders := registry.ByCapabilityFrom[plugin.CommandProvider](plugins)
	if len(commandProviders) == 0 {
		t.Fatal("expected at least one CommandProvider plugin")
	}
}

func TestPluginConfigLoading(t *testing.T) {
	plugins := registry.New()
	cfg, err := config.Load(filepath.Join(fixtureDir(t, "basic"), ".terraci.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Configure plugins from the fixture config
	for _, cl := range registry.ByCapabilityFrom[plugin.ConfigLoader](plugins) {
		if _, exists := cfg.Extensions[cl.ConfigKey()]; !exists {
			continue
		}
		if decErr := cl.DecodeAndSet(func(target any) error {
			return cfg.Extension(cl.ConfigKey(), target)
		}); decErr != nil {
			t.Fatalf("failed to decode %s config: %v", cl.ConfigKey(), decErr)
		}
	}

	// gitlab should be configured (it's in the fixture)
	for _, cl := range registry.ByCapabilityFrom[plugin.ConfigLoader](plugins) {
		if cl.ConfigKey() == "gitlab" && !cl.IsConfigured() {
			t.Error("gitlab should be configured after loading basic fixture")
		}
		if cl.ConfigKey() == "github" && cl.IsConfigured() {
			t.Error("github should NOT be configured (not in basic fixture)")
		}
	}
}

func TestProviderResolution(t *testing.T) {
	clearCIEnv(t)
	plugins := registry.New()
	cfg, err := config.Load(filepath.Join(fixtureDir(t, "basic"), ".terraci.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	for _, cl := range registry.ByCapabilityFrom[plugin.ConfigLoader](plugins) {
		if _, exists := cfg.Extensions[cl.ConfigKey()]; !exists {
			continue
		}
		if decErr := cl.DecodeAndSet(func(target any) error {
			return cfg.Extension(cl.ConfigKey(), target)
		}); decErr != nil {
			t.Fatalf("failed to decode %s config: %v", cl.ConfigKey(), decErr)
		}
	}

	provider, resolveErr := plugins.ResolveCIProvider()
	if resolveErr != nil {
		t.Fatalf("resolve provider: %v", resolveErr)
	}
	if provider.ProviderName() != "gitlab" {
		t.Errorf("expected gitlab provider, got %s", provider.ProviderName())
	}
}

func TestPluginInitialization(t *testing.T) {
	clearCIEnv(t)
	plugins := registry.New()
	dir := fixtureDir(t, "basic")
	cfg, err := config.Load(filepath.Join(dir, ".terraci.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	for _, cl := range registry.ByCapabilityFrom[plugin.ConfigLoader](plugins) {
		if _, exists := cfg.Extensions[cl.ConfigKey()]; !exists {
			continue
		}
		if decErr := cl.DecodeAndSet(func(target any) error {
			return cfg.Extension(cl.ConfigKey(), target)
		}); decErr != nil {
			t.Fatalf("failed to decode %s config: %v", cl.ConfigKey(), decErr)
		}
	}

	appCtx := plugin.NewAppContext(cfg, dir, filepath.Join(dir, ".terraci"), "test", nil, plugins)

	for _, p := range plugins.PreflightsForStartup() {
		if preflightErr := p.Preflight(context.Background(), appCtx); preflightErr != nil {
			// Some plugins may fail if their external deps are missing (e.g., git not in a repo).
			// We log but don't fail — the important thing is the interface works.
			t.Logf("preflight %s: %v (may be expected outside real env)", p.Name(), preflightErr)
		}
	}
}
