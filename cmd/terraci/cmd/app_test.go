package cmd

import "testing"

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
	if secondCtx.Resolver() != app.Plugins {
		t.Fatal("AppContext resolver should point at the fresh command registry")
	}
}
