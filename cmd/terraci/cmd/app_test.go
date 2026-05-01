package cmd

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func TestAppBeginCommandReplacesPluginRegistryAndReusesContextPointer(t *testing.T) {
	app := &App{}

	firstCtx := app.PluginContext()
	firstPlugins := app.Plugins
	firstCtx.Freeze()

	app.BeginCommand()
	secondCtx := app.PluginContext()

	if secondCtx != firstCtx {
		t.Fatal("BeginCommand should keep the stable AppContext pointer used by registered commands")
	}
	if app.Plugins == firstPlugins {
		t.Fatal("BeginCommand should install a fresh plugin registry")
	}
	if secondCtx.IsFrozen() {
		t.Fatal("BeginCommand should reopen AppContext for the next command")
	}
	if _, ok := secondCtx.Resolver().(*registry.Registry); !ok {
		t.Fatalf("AppContext resolver = %T, want *registry.Registry", secondCtx.Resolver())
	}
}
