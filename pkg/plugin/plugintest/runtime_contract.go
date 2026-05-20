package plugintest

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
)

// RuntimeProviderContract describes the fixtures for RuntimeProvider tests.
type RuntimeProviderContract[T any] struct {
	Provider      plugin.RuntimeProvider
	AppContext    *plugin.AppContext
	Context       context.Context
	AssertRuntime func(testing.TB, T)
}

// AssertRuntimeProvider verifies lazy RuntimeProvider construction plus
// RuntimeAs[T] through plugin.BuildRuntime.
func AssertRuntimeProvider[T any](tb testing.TB, c RuntimeProviderContract[T]) {
	tb.Helper()
	if c.Provider == nil {
		tb.Fatal("Provider is nil")
	}
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	appCtx := c.AppContext
	if appCtx == nil {
		appCtx = plugin.NewAppContext(plugin.AppContextOptions{})
	}
	runtime, err := plugin.BuildRuntime[T](ctx, c.Provider, appCtx)
	if err != nil {
		tb.Fatalf("BuildRuntime() error = %v", err)
	}
	if c.AssertRuntime != nil {
		c.AssertRuntime(tb, runtime)
	}
}
