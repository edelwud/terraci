package inmemcache

import (
	"context"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/plugin"
)

func TestCache_SetGet(t *testing.T) {
	cache := newCache()

	if err := cache.Set(context.Background(), "update", "modules", []byte("value"), time.Hour); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, ok, err := cache.Get(context.Background(), "update", "modules")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if string(got) != "value" {
		t.Fatalf("Get() = %q, want %q", string(got), "value")
	}
}

func TestCache_NamespaceIsolation(t *testing.T) {
	cache := newCache()

	if err := cache.Set(context.Background(), "update", "shared", []byte("update"), time.Hour); err != nil {
		t.Fatalf("Set(update) error = %v", err)
	}
	if err := cache.Set(context.Background(), "cost", "shared", []byte("cost"), time.Hour); err != nil {
		t.Fatalf("Set(cost) error = %v", err)
	}

	got, ok, err := cache.Get(context.Background(), "update", "shared")
	if err != nil {
		t.Fatalf("Get(update) error = %v", err)
	}
	if !ok || string(got) != "update" {
		t.Fatalf("Get(update) = (%q, %v), want (%q, true)", string(got), ok, "update")
	}

	got, ok, err = cache.Get(context.Background(), "cost", "shared")
	if err != nil {
		t.Fatalf("Get(cost) error = %v", err)
	}
	if !ok || string(got) != "cost" {
		t.Fatalf("Get(cost) = (%q, %v), want (%q, true)", string(got), ok, "cost")
	}
}

func TestCache_TTLExpiry(t *testing.T) {
	cache := newCache()

	if err := cache.Set(context.Background(), "update", "modules", []byte("value"), 10*time.Millisecond); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	time.Sleep(25 * time.Millisecond)

	_, ok, err := cache.Get(context.Background(), "update", "modules")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok {
		t.Fatal("Get() ok = true after expiry, want false")
	}
}

func TestCache_DeleteAndDeleteNamespace(t *testing.T) {
	cache := newCache()

	if err := cache.Set(context.Background(), "update", "module", []byte("value"), time.Hour); err != nil {
		t.Fatalf("Set(module) error = %v", err)
	}
	if err := cache.Set(context.Background(), "update", "provider", []byte("value"), time.Hour); err != nil {
		t.Fatalf("Set(provider) error = %v", err)
	}

	if err := cache.Delete(context.Background(), "update", "module"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, ok, err := cache.Get(context.Background(), "update", "module"); err != nil {
		t.Fatalf("Get(module) error = %v", err)
	} else if ok {
		t.Fatal("module key should be deleted")
	}

	if err := cache.DeleteNamespace(context.Background(), "update"); err != nil {
		t.Fatalf("DeleteNamespace() error = %v", err)
	}
	if _, ok, err := cache.Get(context.Background(), "update", "provider"); err != nil {
		t.Fatalf("Get(provider) error = %v", err)
	} else if ok {
		t.Fatal("namespace should be deleted")
	}
}

func TestPlugin_EnableByDefaultAndConfigOverride(t *testing.T) {
	p := &Plugin{
		BasePlugin: plugin.BasePlugin[*Config]{
			PluginName: "inmemcache",
			PluginDesc: "Built-in process-local KV cache backend",
			EnableMode: plugin.EnabledByDefault,
			DefaultCfg: func() *Config { return &Config{Enabled: true} },
			IsEnabledFn: func(cfg *Config) bool {
				return cfg == nil || cfg.Enabled
			},
		},
	}

	if !p.IsEnabled() {
		t.Fatal("IsEnabled() = false, want true by default")
	}

	p.SetTypedConfig(&Config{Enabled: false})
	if p.IsEnabled() {
		t.Fatal("IsEnabled() = true after explicit disable, want false")
	}
}
