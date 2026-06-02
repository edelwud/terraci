package plugintest

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
)

// RuntimeBuilderContract describes fixtures for plugin-local lazy runtime tests.
type RuntimeBuilderContract[T any] struct {
	Build         func(context.Context, *plugin.AppContext) (T, error)
	AppContext    *plugin.AppContext
	Context       context.Context
	AssertRuntime func(testing.TB, T)
}

// AssertRuntimeBuilder verifies plugin-local lazy runtime construction.
func AssertRuntimeBuilder[T any](tb testing.TB, c RuntimeBuilderContract[T]) {
	tb.Helper()
	if c.Build == nil {
		tb.Fatal("Build is nil")
	}
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	appCtx := c.AppContext
	if appCtx == nil {
		appCtx = plugin.NewAppContext(plugin.AppContextOptions{})
	}
	runtime, err := c.Build(ctx, appCtx)
	if err != nil {
		tb.Fatalf("runtime builder error = %v", err)
	}
	if c.AssertRuntime != nil {
		c.AssertRuntime(tb, runtime)
	}
}
