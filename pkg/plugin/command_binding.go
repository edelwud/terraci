package plugin

import "fmt"

// CommandInstance returns the command-scoped plugin instance matching name.
func CommandInstance[T Plugin](ctx *AppContext, name string) (T, error) {
	var zero T
	if ctx == nil || ctx.resolver == nil {
		return zero, fmt.Errorf("command plugin %q: plugin context resolver is not bound", name)
	}
	current, ok := ctx.resolver.GetPlugin(name)
	if !ok {
		return zero, fmt.Errorf("command plugin %q: command-scoped instance not found", name)
	}
	typed, ok := current.(T)
	if !ok {
		return zero, fmt.Errorf("command plugin %q: command-scoped instance has type %T", name, current)
	}
	return typed, nil
}
