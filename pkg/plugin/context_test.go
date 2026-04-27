package plugin

import (
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

func TestAppContext_ConfigReturnsCopy(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := NewAppContext(cfg, "/tmp", "/tmp/.terraci", "1.0", nil)

	got := ctx.Config()
	got.ServiceDir = "changed"

	if ctx.Config().ServiceDir != ".terraci" {
		t.Error("Config() should return a defensive copy")
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

func (r contextTestResolver) GetPlugin(string) (Plugin, bool) {
	return r.plugin, r.plugin != nil
}

func (contextTestResolver) ResolveCIProvider() (*ResolvedCIProvider, error) {
	return nil, nil
}

func (contextTestResolver) ResolveChangeDetector() (ChangeDetectionProvider, error) {
	return nil, nil
}

func (contextTestResolver) ResolveKVCacheProvider(string) (KVCacheProvider, error) {
	return nil, nil
}

func (contextTestResolver) ResolveBlobStoreProvider(string) (BlobStoreProvider, error) {
	return nil, nil
}

func (contextTestResolver) CollectContributions(*AppContext) []*pipeline.Contribution {
	return nil
}

func TestAppContext_BeginCommandRebindsResolver(t *testing.T) {
	first := &contextTestPlugin{name: "cmd"}
	second := &contextTestPlugin{name: "cmd"}
	ctx := NewAppContext(nil, "/tmp", "/tmp/.terraci", "1.0", nil, contextTestResolver{plugin: first})

	if got := CommandPlugin(ctx, &contextTestPlugin{name: "cmd"}); got != first {
		t.Fatalf("CommandPlugin() = %p, want first %p", got, first)
	}

	ctx.Freeze()
	ctx.BeginCommand(contextTestResolver{plugin: second})
	if ctx.IsFrozen() {
		t.Fatal("BeginCommand should reopen frozen context")
	}
	if got := CommandPlugin(ctx, &contextTestPlugin{name: "cmd"}); got != second {
		t.Fatalf("CommandPlugin() = %p, want second %p", got, second)
	}

	ctx.Update(nil, "/next", "/next/.terraci", "2.0")
	if ctx.WorkDir() != "/next" {
		t.Fatalf("WorkDir() = %q, want /next", ctx.WorkDir())
	}
}
