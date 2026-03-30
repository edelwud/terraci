package update

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

func TestPlugin_Name(t *testing.T) {
	p := newTestPlugin(t)
	if p.Name() != "update" {
		t.Errorf("Name() = %q, want %q", p.Name(), "update")
	}
	if p.Description() == "" {
		t.Error("Description() is empty")
	}
}

func TestPlugin_EnablePolicy(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *updateengine.UpdateConfig
		setCfg         bool
		wantConfigured bool
		wantEnabled    bool
	}{
		{
			name:           "no config set",
			setCfg:         false,
			wantConfigured: false,
			wantEnabled:    false,
		},
		{
			name:           "config set, enabled=false",
			cfg:            &updateengine.UpdateConfig{Enabled: false},
			setCfg:         true,
			wantConfigured: true,
			wantEnabled:    false,
		},
		{
			name:           "config set, enabled=true",
			cfg:            &updateengine.UpdateConfig{Enabled: true},
			setCfg:         true,
			wantConfigured: true,
			wantEnabled:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPlugin(t)
			if tt.setCfg {
				enablePlugin(t, p, tt.cfg)
			}
			if got := p.IsConfigured(); got != tt.wantConfigured {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.wantConfigured)
			}
			if got := p.IsEnabled(); got != tt.wantEnabled {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.wantEnabled)
			}
		})
	}
}

func TestPlugin_Preflight_Disabled(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if p.registryFactory != nil {
		t.Error("registry factory should be nil when plugin is not configured")
	}
}

func TestPlugin_Preflight_ConfiguredButDisabled(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: false})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if p.registryFactory != nil {
		t.Error("registry factory should be nil when plugin is configured but disabled")
	}
}

func TestPlugin_Preflight_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if p.registryFactory != nil {
		t.Fatal("registry factory should remain nil after preflight; runtime is lazy")
	}
}

func TestPlugin_Preflight_InvalidConfig(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{
		Enabled: true,
		Target:  "invalid-target",
	})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err == nil {
		t.Fatal("Preflight() error = nil, want invalid configuration error")
	}
}

func TestPlugin_Reset(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{})

	p.Reset()

	if p.IsConfigured() {
		t.Error("IsConfigured() should be false after Reset")
	}
	if p.registryFactory != nil {
		t.Error("registry factory should be nil after Reset")
	}
}

func TestPlugin_Runtime_CreatesRegistryLazily(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{})

	runtime := plugintest.MustRuntime[*updateRuntime](t, p, newTestAppContext(t, t.TempDir()))
	if runtime.registry == nil {
		t.Fatal("runtime.registry should not be nil")
	}
}
