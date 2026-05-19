package plugin

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// CommandInstance returns the command-scoped plugin instance matching name.
func CommandInstance[T Plugin](ctx *AppContext, name string) (T, error) {
	var zero T
	if ctx == nil {
		return zero, fmt.Errorf("command plugin %q: plugin context is not bound", name)
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

// CommandPlugin resolves the command-scoped AppContext and plugin instance for
// a cobra callback. It is the preferred command boundary for built-in and
// external plugins.
func CommandPlugin[T Plugin](cmd *cobra.Command, name string) (*AppContext, T, error) {
	var zero T
	if cmd == nil {
		return nil, zero, fmt.Errorf("command plugin %q: cobra command is nil", name)
	}
	appCtx := FromContext(cmd.Context())
	current, err := CommandInstance[T](appCtx, name)
	if err != nil {
		return appCtx, zero, err
	}
	return appCtx, current, nil
}

// RequireEnabled returns message when p reports disabled.
func RequireEnabled(p interface{ IsEnabled() bool }, message string) error {
	if message == "" {
		message = "plugin is not enabled"
	}
	if p == nil {
		return errors.New(message)
	}
	if !p.IsEnabled() {
		return errors.New(message)
	}
	return nil
}
