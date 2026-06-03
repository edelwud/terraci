package test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func TestPluginRegistration(t *testing.T) {
	plugins := registry.New()
	inventory := plugins.Inventory().Plugins()
	if len(inventory) != 10 {
		t.Fatalf("expected 10 plugins, got %d", len(inventory))
	}

	names := make(map[string]bool)
	for _, p := range inventory {
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
	schemas := plugins.ExtensionSchemas()
	if len(schemas) == 0 {
		t.Fatal("expected at least one extension schema")
	}

	configKeys := make(map[string]bool)
	for key := range schemas {
		if key == "" {
			t.Error("extension schema has empty key")
		}
		if configKeys[key] {
			t.Errorf("duplicate extension schema key: %s", key)
		}
		configKeys[key] = true
	}

	for _, expectedKey := range []string{"gitlab", "github", "cost", "diskblob", "inmemcache", "policy"} {
		if !configKeys[expectedKey] {
			t.Errorf("missing expected ConfigKey: %s", expectedKey)
		}
	}

	initSnapshot, err := plugins.InitWizardSnapshot()
	if err != nil {
		t.Fatalf("InitWizardSnapshot() error = %v", err)
	}
	if len(initSnapshot.ProviderOptions()) < 2 {
		t.Fatalf("expected at least 2 init provider options (gitlab, github), got %d", len(initSnapshot.ProviderOptions()))
	}

	if len(plugins.Commands()) == 0 {
		t.Fatal("expected at least one plugin command")
	}
}

func TestPluginConfigLoading(t *testing.T) {
	plugins := registry.New()
	cfg, err := config.Load(filepath.Join(fixtureDir(t, "basic"), ".terraci.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	configurePluginsFromConfig(t, plugins, cfg)

	provider, err := plugins.ResolveCIProvider()
	if err != nil {
		t.Fatalf("ResolveCIProvider() after gitlab config error = %v", err)
	}
	if provider.ProviderName() != "gitlab" {
		t.Fatalf("ResolveCIProvider() = %q, want gitlab", provider.ProviderName())
	}

	t.Setenv("TERRACI_PROVIDER", "github")
	if _, err := plugins.ResolveCIProvider(); err == nil {
		t.Fatal("ResolveCIProvider(github) error = nil, want disabled/unconfigured provider error")
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

	if preflightErr := plugins.RunPreflight(context.Background(), appCtx); preflightErr != nil {
		if !strings.Contains(preflightErr.Error(), "preflight plugin git") {
			t.Fatalf("RunPreflight() error = %v", preflightErr)
		}
		t.Logf("preflight git: %v (may be expected outside real env)", preflightErr)
	}
}

func configurePluginsFromConfig(t *testing.T, plugins *registry.Registry, cfg *config.Config) {
	t.Helper()
	if err := plugins.DecodeConfig(cfg); err != nil {
		t.Fatalf("failed to decode plugin config: %v", err)
	}
}
