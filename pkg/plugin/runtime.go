package plugin

import (
	"context"
	"fmt"
)

// RuntimeProvider is the preferred pattern for plugins with heavy command-time
// setup. Runtime creation is lazy and command-driven; the framework does not
// invoke it automatically during startup.
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
// plugin-local type.
func RuntimeAs[T any](runtime any) (T, error) {
	typed, ok := runtime.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("unexpected runtime type %T", runtime)
	}
	return typed, nil
}
