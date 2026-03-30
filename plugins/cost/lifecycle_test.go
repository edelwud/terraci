package cost

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
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
		cfg            *model.CostConfig
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
			cfg:            &model.CostConfig{Enabled: false},
			setCfg:         true,
			wantConfigured: true,
			wantEnabled:    false,
		},
		{
			name:           "config set, enabled=true",
			cfg:            &model.CostConfig{Enabled: true},
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
}

func TestPlugin_Initialize_ConfiguredButDisabled(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{Enabled: false, CacheDir: t.TempDir()})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
}

func TestPlugin_Initialize_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	cacheDir := t.TempDir()
	enablePlugin(t, p, &model.CostConfig{
		Enabled:  true,
		CacheDir: cacheDir,
	})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
}

func TestPlugin_Initialize_InvalidTTL(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled:  true,
		CacheDir: t.TempDir(),
		CacheTTL: "not-a-duration",
	})
	appCtx := newTestAppContext(t, t.TempDir())

	err := p.Initialize(context.Background(), appCtx)
	if err == nil {
		t.Fatal("Initialize() error = nil, want invalid configuration error")
	}
	if got := err.Error(); got == "" || got == "invalid cost configuration" {
		t.Fatalf("Initialize() error = %q, want actionable validation error", got)
	}
}

func TestPlugin_Reset(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled:  true,
		CacheDir: t.TempDir(),
	})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	p.Reset()

	if p.IsConfigured() {
		t.Error("IsConfigured() should be false after Reset")
	}
}
