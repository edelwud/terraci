package plugin

import (
	"context"
	"fmt"
)

// RuntimeProvider is the preferred pattern for plugins with heavy command-time
// setup. Runtime creation is lazy and command-driven; the framework does not
// invoke it automatically during startup or preflight.
//
// Use Preflightable for cheap validation and environment checks. Use
// RuntimeProvider for typed runtime construction inside plugin commands and
// use-cases.
//
// Typical shape:
//
//	func (p *Plugin) Runtime(_ context.Context, appCtx *AppContext) (any, error) {
//		return newRuntime(appCtx, p.Config(), runtimeOptions{})
//	}
//
//	func (p *Plugin) runtime(ctx context.Context, appCtx *AppContext, opts runtimeOptions) (*myRuntime, error) {
//		if opts == (runtimeOptions{}) {
//			rawRuntime, err := p.Runtime(ctx, appCtx)
//			if err != nil {
//				return nil, err
//			}
//			return RuntimeAs[*myRuntime](rawRuntime)
//		}
//		return newRuntime(appCtx, p.Config(), opts)
//	}
type RuntimeProvider interface {
	Plugin
	Runtime(ctx context.Context, appCtx *AppContext) (any, error)
}

// RuntimeAs converts a runtime returned by RuntimeProvider into the expected
// plugin-local type at the plugin boundary. The framework intentionally treats
// runtime values as opaque.
func RuntimeAs[T any](runtime any) (T, error) {
	typed, ok := runtime.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("unexpected runtime type %T", runtime)
	}
	return typed, nil
}

// BuildRuntime calls p.Runtime and type-asserts the result to T in one step.
// It combines the Runtime() call and RuntimeAs[T]() assertion — the recommended
// shorthand for plugin use-cases:
//
//	func (p *Plugin) runtime(ctx context.Context, appCtx *AppContext) (*myRuntime, error) {
//		return plugin.BuildRuntime[*myRuntime](ctx, p, appCtx)
//	}
func BuildRuntime[T any](ctx context.Context, p RuntimeProvider, appCtx *AppContext) (T, error) {
	raw, err := p.Runtime(ctx, appCtx)
	if err != nil {
		var zero T
		return zero, err
	}
	return RuntimeAs[T](raw)
}
