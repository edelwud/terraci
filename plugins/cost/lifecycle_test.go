package cost

import (
	"context"
	"testing"

	costengine "github.com/edelwud/terraci/plugins/cost/internal"
)

func TestPlugin_Name(t *testing.T) {
	p := newTestPlugin(t)
	if p.Name() != "cost" {
		t.Errorf("Name() = %q, want %q", p.Name(), "cost")
	}
	if p.Description() == "" {
		t.Error("Description() is empty")
	}
}

func TestPlugin_EnablePolicy(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *costengine.CostConfig
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
			cfg:            &costengine.CostConfig{Enabled: false},
			setCfg:         true,
			wantConfigured: true,
			wantEnabled:    false,
		},
		{
			name:           "config set, enabled=true",
			cfg:            &costengine.CostConfig{Enabled: true},
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
	if p.getEstimator() != nil {
		t.Error("estimator should be nil when plugin is not configured")
	}
}

func TestPlugin_Initialize_ConfiguredButDisabled(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &costengine.CostConfig{Enabled: false, CacheDir: t.TempDir()})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if p.getEstimator() != nil {
		t.Error("estimator should be nil when plugin is configured but disabled")
	}
}

func TestPlugin_Initialize_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	cacheDir := t.TempDir()
	enablePlugin(t, p, &costengine.CostConfig{
		Enabled:  true,
		CacheDir: cacheDir,
	})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if p.getEstimator() == nil {
		t.Fatal("estimator should not be nil after Initialize with enabled config")
	}
	if p.serviceDirRel != appCtx.Config().ServiceDir {
		t.Errorf("serviceDirRel = %q, want %q", p.serviceDirRel, appCtx.Config().ServiceDir)
	}
}

func TestPlugin_Initialize_InvalidTTL(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &costengine.CostConfig{
		Enabled:  true,
		CacheDir: t.TempDir(),
		CacheTTL: "not-a-duration",
	})
	appCtx := newTestAppContext(t, t.TempDir())

	// Should not return error — invalid TTL is logged as warning, not fatal
	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if p.getEstimator() == nil {
		t.Fatal("estimator should still be created with invalid TTL")
	}
}

func TestPlugin_Reset(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &costengine.CostConfig{
		Enabled:  true,
		CacheDir: t.TempDir(),
	})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	// Verify state is set before reset
	if p.getEstimator() == nil {
		t.Fatal("estimator should be set before reset")
	}

	p.Reset()

	if p.IsConfigured() {
		t.Error("IsConfigured() should be false after Reset")
	}
	if p.getEstimator() != nil {
		t.Error("estimator should be nil after Reset")
	}
	if p.serviceDirRel != "" {
		t.Errorf("serviceDirRel = %q, want empty after Reset", p.serviceDirRel)
	}
}
