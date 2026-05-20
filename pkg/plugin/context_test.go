package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestAppContext_Accessors(t *testing.T) {
	cfg := config.DefaultConfig()
	r := ci.NewMemoryReportStore()
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

func TestAppContext_NoReportsCreatesDefaultStore(t *testing.T) {
	ctx := NewAppContext(AppContextOptions{})
	if ctx.Reports() == nil {
		t.Fatal("Reports() should not be nil for empty options")
	}
}

func TestAppContext_PipelineContributionsAreSnapshot(t *testing.T) {
	contrib := &pipeline.Contribution{Jobs: []pipeline.ContributedJob{{Name: "summary"}}}
	ctx := NewAppContext(AppContextOptions{
		PipelineContributions: []*pipeline.Contribution{contrib},
	})

	got := ctx.PipelineContributions()
	if len(got) != 1 || got[0] != contrib {
		t.Fatalf("PipelineContributions() = %#v, want original contribution pointer", got)
	}
	got[0] = nil
	if again := ctx.PipelineContributions(); len(again) != 1 || again[0] != contrib {
		t.Fatalf("PipelineContributions() was mutated through returned slice: %#v", again)
	}
}

type contextTestPlugin struct {
	name    string
	enabled bool
}

func (p *contextTestPlugin) Name() string        { return p.name }
func (p *contextTestPlugin) Description() string { return p.name }
func (p *contextTestPlugin) IsEnabled() bool     { return p.enabled }

type contextTestResolver struct {
	NoopResolver
	plugin Plugin
}

func (r contextTestResolver) GetPlugin(string) (Plugin, bool) {
	return r.plugin, r.plugin != nil
}

func TestCommandPluginLookup_LooksUpFromResolver(t *testing.T) {
	target := &contextTestPlugin{name: "cmd"}
	ctx := NewAppContext(AppContextOptions{
		Resolver: contextTestResolver{plugin: target},
	})

	got, err := commandInstance[*contextTestPlugin](ctx, "cmd")
	if err != nil {
		t.Fatalf("commandInstance() error = %v", err)
	}
	if got != target {
		t.Fatalf("commandInstance() = %p, want %p", got, target)
	}
}

func TestCommandPluginLookup_RejectsNilContext(t *testing.T) {
	if _, err := commandInstance[*contextTestPlugin](nil, "cmd"); err == nil {
		t.Fatal("commandInstance(nil) error = nil, want missing context error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingMissingContext)
	}
}

func TestCommandPluginLookup_RejectsMissingCommandLookup(t *testing.T) {
	ctx := &AppContext{}
	if _, err := commandInstance[*contextTestPlugin](ctx, "cmd"); err == nil {
		t.Fatal("commandInstance() error = nil, want missing command lookup error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingMissingLookup)
	}
}

func TestCommandPlugin_ReturnsContextAndPlugin(t *testing.T) {
	target := &contextTestPlugin{name: "cmd", enabled: true}
	appCtx := NewAppContext(AppContextOptions{
		Resolver: contextTestResolver{plugin: target},
	})
	cmd := &cobra.Command{}
	cmd.SetContext(WithContext(context.Background(), appCtx))

	gotCtx, gotPlugin, err := CommandPlugin[*contextTestPlugin](cmd, "cmd")
	if err != nil {
		t.Fatalf("CommandPlugin() error = %v", err)
	}
	if gotCtx != appCtx {
		t.Fatalf("CommandPlugin() ctx = %p, want %p", gotCtx, appCtx)
	}
	if gotPlugin != target {
		t.Fatalf("CommandPlugin() plugin = %p, want %p", gotPlugin, target)
	}
}

func TestCommandPlugin_RejectsNilCommand(t *testing.T) {
	if _, _, err := CommandPlugin[*contextTestPlugin](nil, "cmd"); err == nil {
		t.Fatal("CommandPlugin(nil) error = nil, want error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingNilCommand)
	}
}

func TestCommandPlugin_RejectsMissingAppContext(t *testing.T) {
	cmd := &cobra.Command{}
	if _, _, err := CommandPlugin[*contextTestPlugin](cmd, "cmd"); err == nil {
		t.Fatal("CommandPlugin() error = nil, want missing app context error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingMissingContext)
	}
}

func TestCommandPlugin_RejectsMissingCommandLookup(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(WithContext(context.Background(), NewAppContext(AppContextOptions{})))

	if _, _, err := CommandPlugin[*contextTestPlugin](cmd, "cmd"); err == nil {
		t.Fatal("CommandPlugin() error = nil, want missing command lookup error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingMissingLookup)
	}
}

func TestCommandPlugin_RejectsWrongPluginType(t *testing.T) {
	appCtx := NewAppContext(AppContextOptions{
		Resolver: contextTestResolver{plugin: &contextTestPlugin{name: "cmd"}},
	})
	cmd := &cobra.Command{}
	cmd.SetContext(WithContext(context.Background(), appCtx))

	if _, _, err := CommandPlugin[*otherContextTestPlugin](cmd, "cmd"); err == nil {
		t.Fatal("CommandPlugin() error = nil, want wrong type error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingWrongType)
	}
}

func TestRequireEnabled(t *testing.T) {
	if err := RequireEnabled(&contextTestPlugin{enabled: true}, "disabled"); err != nil {
		t.Fatalf("RequireEnabled(enabled) error = %v", err)
	}
	if err := RequireEnabled(&contextTestPlugin{}, "disabled"); err == nil || err.Error() != "disabled" {
		t.Fatalf("RequireEnabled(disabled) error = %v, want disabled", err)
	} else {
		var disabled *DisabledPluginError
		if !errors.As(err, &disabled) {
			t.Fatalf("RequireEnabled(disabled) error type = %T, want DisabledPluginError", err)
		}
	}
	if err := RequireEnabled(nil, "missing"); err == nil || err.Error() != "missing" {
		t.Fatalf("RequireEnabled(nil) error = %v, want missing", err)
	}
}

type otherContextTestPlugin struct{}

func (p *otherContextTestPlugin) Name() string        { return "other" }
func (p *otherContextTestPlugin) Description() string { return "other" }

func TestCommandContextBinding_RoundTrips(t *testing.T) {
	appCtx := NewAppContext(AppContextOptions{Version: "v"})
	carrier := WithContext(context.Background(), appCtx)
	got := fromContext(carrier)
	if got != appCtx {
		t.Fatalf("fromContext() = %p, want %p", got, appCtx)
	}
	if fromContext(context.Background()) != nil {
		t.Fatal("fromContext on empty context should be nil")
	}
	if fromContext(context.TODO()) != nil {
		// context.TODO is the canonical "no-value" Context; fromContext
		// must still return nil for a context that has no AppContext key
		// attached.
		t.Fatal("fromContext(empty context) should be nil")
	}
}

func TestAppContextFromCommand(t *testing.T) {
	appCtx := NewAppContext(AppContextOptions{Version: "v"})
	cmd := &cobra.Command{}
	cmd.SetContext(WithContext(context.Background(), appCtx))

	got, err := AppContextFromCommand(cmd)
	if err != nil {
		t.Fatalf("AppContextFromCommand() error = %v", err)
	}
	if got != appCtx {
		t.Fatalf("AppContextFromCommand() = %p, want %p", got, appCtx)
	}
}

func assertCommandBindingReason(t *testing.T, err error, reason CommandBindingReason) {
	t.Helper()

	var bindingErr *CommandBindingError
	if !errors.As(err, &bindingErr) {
		t.Fatalf("error type = %T, want CommandBindingError", err)
	}
	if bindingErr.Reason != reason {
		t.Fatalf("CommandBindingError.Reason = %q, want %q", bindingErr.Reason, reason)
	}
}
