package update

import (
	"context"
	"testing"

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

func TestPlugin_Initialize_Disabled(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if p.registry != nil {
		t.Error("registry should be nil when plugin is not configured")
	}
}

func TestPlugin_Initialize_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if p.registry == nil {
		t.Fatal("registry should not be nil after Initialize with enabled config")
	}
	if p.serviceDirRel != appCtx.Config.ServiceDir {
		t.Errorf("serviceDirRel = %q, want %q", p.serviceDirRel, appCtx.Config.ServiceDir)
	}
}

func TestPlugin_Initialize_InvalidConfig(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{
		Enabled: true,
		Target:  "invalid-target",
	})
	appCtx := newTestAppContext(t, t.TempDir())

	// Should not return error — invalid config is logged as warning, not fatal.
	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if p.registry == nil {
		t.Fatal("registry should still be created despite invalid config")
	}
}

func TestPlugin_Reset(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if p.registry == nil {
		t.Fatal("registry should be set before reset")
	}

	p.Reset()

	if p.IsConfigured() {
		t.Error("IsConfigured() should be false after Reset")
	}
	if p.registry != nil {
		t.Error("registry should be nil after Reset")
	}
	if p.serviceDirRel != "" {
		t.Errorf("serviceDirRel = %q, want empty after Reset", p.serviceDirRel)
	}
}
