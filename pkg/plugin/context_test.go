package plugin

import (
	"errors"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestAppContext_Freeze(t *testing.T) {
	ctx := NewAppContext(nil, "/tmp", "/tmp/.terraci", "1.0", nil)

	if ctx.IsFrozen() {
		t.Error("should not be frozen initially")
	}

	ctx.Freeze()

	if !ctx.IsFrozen() {
		t.Error("should be frozen after Freeze()")
	}

	ctx.Update(nil, "/other", "/other/.terraci", "2.0")
	if ctx.WorkDir() != "/tmp" {
		t.Error("frozen context should ignore framework updates")
	}
}

func TestAppContext_ReportRegistry(t *testing.T) {
	r := NewReportRegistry()
	ctx := NewAppContext(nil, "", "", "", r)

	if ctx.Reports() == nil {
		t.Fatal("Reports should not be nil")
	}

	// Verify it's the same registry
	if ctx.Reports() != r {
		t.Error("Reports should be the same registry instance")
	}
}

func TestAppContext_ConfigReturnsBoundPointer(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := NewAppContext(cfg, "/tmp", "/tmp/.terraci", "1.0", nil)

	if ctx.Config() != cfg {
		t.Error("Config() should return the bound configuration pointer")
	}
}

type contextTestPlugin struct {
	name string
}

func (p *contextTestPlugin) Name() string        { return p.name }
func (p *contextTestPlugin) Description() string { return p.name }

type contextTestResolver struct {
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

func (contextTestResolver) ResolveCIProvider() (*ResolvedCIProvider, error) {
	return nil, errors.New("not configured")
}

func (contextTestResolver) ResolveChangeDetector() (ChangeDetectionProvider, error) {
	return nil, errors.New("not configured")
}

func (contextTestResolver) ResolveKVCacheProvider(string) (KVCacheProvider, error) {
	return nil, errors.New("not configured")
}

func (contextTestResolver) ResolveBlobStoreProvider(string) (BlobStoreProvider, error) {
	return nil, errors.New("not configured")
}

func (contextTestResolver) CollectContributions(*AppContext) []*pipeline.Contribution {
	return nil
}

func (contextTestResolver) PreflightsForStartup() []Preflightable { return nil }

func TestAppContext_BeginCommandRebindsResolver(t *testing.T) {
	first := &contextTestPlugin{name: "cmd"}
	second := &contextTestPlugin{name: "cmd"}
	ctx := NewAppContext(nil, "/tmp", "/tmp/.terraci", "1.0", nil, contextTestResolver{plugin: first})

	got, err := CommandInstance[*contextTestPlugin](ctx, "cmd")
	if err != nil {
		t.Fatalf("CommandInstance() error = %v", err)
	}
	if got != first {
		t.Fatalf("CommandInstance() = %p, want first %p", got, first)
	}

	ctx.Freeze()
	ctx.BeginCommand(contextTestResolver{plugin: second})
	if ctx.IsFrozen() {
		t.Fatal("BeginCommand should reopen frozen context")
	}
	got, err = CommandInstance[*contextTestPlugin](ctx, "cmd")
	if err != nil {
		t.Fatalf("CommandInstance() error = %v", err)
	}
	if got != second {
		t.Fatalf("CommandInstance() = %p, want second %p", got, second)
	}

	ctx.Update(nil, "/next", "/next/.terraci", "2.0")
	if ctx.WorkDir() != "/next" {
		t.Fatalf("WorkDir() = %q, want /next", ctx.WorkDir())
	}
}

func TestCommandInstanceRejectsMissingResolver(t *testing.T) {
	ctx := NewAppContext(nil, "/tmp", "/tmp/.terraci", "1.0", nil)
	ctx.resolver = nil

	if _, err := CommandInstance[*contextTestPlugin](ctx, "cmd"); err == nil {
		t.Fatal("CommandInstance() error = nil, want missing resolver error")
	}
}
