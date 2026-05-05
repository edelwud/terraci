package plugin

import "fmt"

// CommandInstance returns the command-scoped plugin instance matching name.
func CommandInstance[T Plugin](ctx *AppContext, name string) (T, error) {
	var zero T
	if ctx == nil {
		return zero, fmt.Errorf("command plugin %q: plugin context resolver is not bound", name)
	}
	if ctx.commands == nil {
		return zero, fmt.Errorf("command plugin %q: command lookup is not bound", name)
	}
	current, ok := ctx.commands.GetPlugin(name)
	if !ok {
		return zero, fmt.Errorf("command plugin %q: command-scoped instance not found", name)
	}
	typed, ok := current.(T)
	if !ok {
		return zero, fmt.Errorf("command plugin %q: command-scoped instance has type %T (want %T)", name, current, zero)
	}
	return typed, nil
}
