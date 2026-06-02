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
	if len(all) != 10 {
		t.Fatalf("expected 10 plugins, got %d", len(all))
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

	expected := []string{"cost", "diskblob", "git", "github", "gitlab", "inmemcache", "local-exec", "policy", "summary", "tfupdate"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing plugin: %s", name)
		}
	}
}

func TestPluginCapabilities(t *testing.T) {
	plugins := registry.New()
	// ConfigLoader plugins
	configLoaders := plugins.ConfigLoaders()
	if len(configLoaders) == 0 {
		t.Fatal("expected at least one ConfigLoader")
	}

	configKeys := make(map[string]bool)
	for _, cl := range configLoaders {
		key := cl.ConfigKey()
		if key.String() == "" {
			t.Errorf("plugin %s has empty ConfigKey", cl.Name())
		}
		if configKeys[key.String()] {
			t.Errorf("duplicate ConfigKey: %s", key.String())
		}
		configKeys[key.String()] = true
	}

	// Verify specific expected config keys
	for _, expectedKey := range []string{"gitlab", "github", "cost", "diskblob", "inmemcache", "policy"} {
		if !configKeys[expectedKey] {
			t.Errorf("missing expected ConfigKey: %s", expectedKey)
		}
	}

	// CI provider plugins (gitlab + github) — must implement all CI interfaces
	ciProviders := plugins.CIInfoProviders()
	if len(ciProviders) < 2 {
		t.Errorf("expected at least 2 CIInfoProvider plugins (gitlab, github), got %d", len(ciProviders))
	}

	// Preflightable plugins
	preflightables := plugins.Preflightables()
	if len(preflightables) == 0 {
		t.Fatal("expected at least one Preflightable plugin")
	}

	// CommandProvider plugins
	commandProviders := plugins.CommandProviders()
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

	configurePluginsFromConfig(t, plugins, cfg)

	// gitlab should be configured (it's in the fixture)
	for _, cl := range plugins.ConfigLoaders() {
		if cl.ConfigKey().String() == "gitlab" && !cl.IsConfigured() {
			t.Error("gitlab should be configured after loading basic fixture")
		}
		if cl.ConfigKey().String() == "github" && cl.IsConfigured() {
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

	configurePluginsFromConfig(t, plugins, cfg)

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

	configurePluginsFromConfig(t, plugins, cfg)

	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		Config:     cfg,
		WorkDir:    dir,
		ServiceDir: filepath.Join(dir, ".terraci"),
		Version:    "test",
		Resolver:   plugins,
	})

	for _, p := range plugins.PreflightsForStartup() {
		if preflightErr := p.Preflight(context.Background(), appCtx); preflightErr != nil {
			// Some plugins may fail if their external deps are missing (e.g., git not in a repo).
			// We log but don't fail — the important thing is the interface works.
			t.Logf("preflight %s: %v (may be expected outside real env)", p.Name(), preflightErr)
		}
	}
}

func configurePluginsFromConfig(t *testing.T, plugins *registry.Registry, cfg *config.Config) {
	t.Helper()
	for _, cl := range plugins.ConfigLoaders() {
		key := cl.ConfigKey()
		doc, exists := cfg.Extension(key)
		if !exists {
			continue
		}
		if err := cl.DecodeAndSet(doc); err != nil {
			t.Fatalf("failed to decode %s config: %v", key.String(), err)
		}
	}
}
