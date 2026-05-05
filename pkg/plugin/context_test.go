package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestAppContext_Accessors(t *testing.T) {
	cfg := config.DefaultConfig()
	r := NewReportRegistry()
	ctx := NewAppContext(AppContextOptions{
		Config:     cfg,
		WorkDir:    "/tmp",
		ServiceDir: "/tmp/.terraci",
		Version:    "1.0",
		Reports:    r,
	})

	if ctx.Config() != cfg {
		t.Error("Config() should return the bound configuration pointer")
	}
	if ctx.WorkDir() != "/tmp" {
		t.Errorf("WorkDir() = %q, want /tmp", ctx.WorkDir())
	}
	if ctx.ServiceDir() != "/tmp/.terraci" {
		t.Errorf("ServiceDir() = %q, want /tmp/.terraci", ctx.ServiceDir())
	}
	if ctx.Version() != "1.0" {
		t.Errorf("Version() = %q, want 1.0", ctx.Version())
	}
	if ctx.Reports() != r {
		t.Error("Reports() should be the same registry instance")
	}
	if ctx.Resolver() == nil {
		t.Error("Resolver() should never return nil")
	}
}

func TestAppContext_NoResolverFallsBackToNoop(t *testing.T) {
	ctx := NewAppContext(AppContextOptions{})
	if ctx.Resolver() == nil {
		t.Fatal("Resolver() should never return nil")
	}
	if _, err := ctx.Resolver().ResolveCIProvider(); err == nil {
		t.Error("noop resolver should reject ResolveCIProvider")
	}
	if !errors.Is(errFromCallSite(ctx.Resolver().ResolveCIProvider), ErrNoResolver) {
		t.Error("noop resolver should return ErrNoResolver")
	}
}

func errFromCallSite[T any](fn func() (T, error)) error {
	_, err := fn()
	return err
}

func TestAppContext_NoReportsCreatesDefaultRegistry(t *testing.T) {
	ctx := NewAppContext(AppContextOptions{})
	if ctx.Reports() == nil {
		t.Fatal("Reports() should not be nil for empty options")
	}
}

type contextTestPlugin struct {
	name string
}

func (p *contextTestPlugin) Name() string        { return p.name }
func (p *contextTestPlugin) Description() string { return p.name }

type contextTestResolver struct {
	NoopResolver
	plugin Plugin
}

func (r contextTestResolver) All() []Plugin {
	if r.plugin == nil {
		return nil
	}
	return []Plugin{r.plugin}
}

func (r contextTestResolver) GetPlugin(string) (Plugin, bool) {
	return r.plugin, r.plugin != nil
}

func TestCommandInstance_LooksUpFromResolver(t *testing.T) {
	target := &contextTestPlugin{name: "cmd"}
	ctx := NewAppContext(AppContextOptions{
		Resolver: contextTestResolver{plugin: target},
	})

	got, err := CommandInstance[*contextTestPlugin](ctx, "cmd")
	if err != nil {
		t.Fatalf("CommandInstance() error = %v", err)
	}
	if got != target {
		t.Fatalf("CommandInstance() = %p, want %p", got, target)
	}
}

func TestCommandInstance_RejectsNilContext(t *testing.T) {
	if _, err := CommandInstance[*contextTestPlugin](nil, "cmd"); err == nil {
		t.Fatal("CommandInstance(nil) error = nil, want missing context error")
	}
}

func TestCommandInstance_RejectsNilResolver(t *testing.T) {
	ctx := &AppContext{}
	if _, err := CommandInstance[*contextTestPlugin](ctx, "cmd"); err == nil {
		t.Fatal("CommandInstance() error = nil, want missing resolver error")
	}
}

func TestWithFromContext_RoundTrips(t *testing.T) {
	appCtx := NewAppContext(AppContextOptions{Version: "v"})
	carrier := WithContext(context.Background(), appCtx)
	got := FromContext(carrier)
	if got != appCtx {
		t.Fatalf("FromContext() = %p, want %p", got, appCtx)
	}
	if FromContext(context.Background()) != nil {
		t.Fatal("FromContext on empty context should be nil")
	}
	if FromContext(context.TODO()) != nil {
		// context.TODO is the canonical "no-value" Context; FromContext
		// must still return nil for a context that has no AppContext key
		// attached.
		t.Fatal("FromContext(empty context) should be nil")
	}
}

// Sanity: NoopResolver's CollectContributions and PreflightsForStartup
// return nil so callers don't need nil-checks before iterating.
func TestNoopResolver_LifecycleHooks(t *testing.T) {
	var r NoopResolver
	if got := r.CollectContributions(nil); got != nil {
		t.Errorf("CollectContributions = %v, want nil", got)
	}
	if got := r.PreflightsForStartup(); got != nil {
		t.Errorf("PreflightsForStartup = %v, want nil", got)
	}

	// Use pipeline import to keep it referenced — guards against future
	// drift where the resolver no longer exposes pipeline types.
	_ = []*pipeline.Contribution(nil)
}
