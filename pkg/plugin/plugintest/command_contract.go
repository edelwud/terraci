package plugintest

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/plugin"
)

// StaticCommandLookup is a minimal command-scoped plugin lookup for
// CommandPlugin contract tests.
type StaticCommandLookup map[string]plugin.Plugin

// GetPlugin implements plugin.CommandLookup.
func (l StaticCommandLookup) GetPlugin(name string) (plugin.Plugin, bool) {
	p, ok := l[name]
	return p, ok
}

// CommandBindingContract describes the fixtures for CommandPlugin[T].
type CommandBindingContract[T plugin.Plugin] struct {
	Name           string
	Plugin         T
	WrongPlugin    plugin.Plugin
	AssertResolved func(testing.TB, T)
}

// AssertCommandBinding verifies command-scoped lookup, missing context/lookup,
// wrong type, and typed CommandBindingError reasons for CommandPlugin[T].
func AssertCommandBinding[T plugin.Plugin](tb testing.TB, c CommandBindingContract[T]) {
	tb.Helper()
	if c.Name == "" {
		tb.Fatal("Name is empty")
	}
	if any(c.Plugin) == nil {
		tb.Fatal("Plugin is nil")
	}

	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		CommandLookup: StaticCommandLookup{c.Name: c.Plugin},
	})
	cmd := &cobra.Command{}
	cmd.SetContext(plugin.WithContext(context.Background(), appCtx))
	gotCtx, got, err := plugin.CommandPlugin[T](cmd, c.Name)
	if err != nil {
		tb.Fatalf("CommandPlugin(success) error = %v", err)
	}
	if gotCtx != appCtx {
		tb.Fatalf("CommandPlugin appCtx = %p, want %p", gotCtx, appCtx)
	}
	if c.AssertResolved != nil {
		c.AssertResolved(tb, got)
	}

	assertCommandBindingReason[T](tb, nil, c.Name, plugin.CommandBindingNilCommand)

	missingContext := &cobra.Command{}
	assertCommandBindingReason[T](tb, missingContext, c.Name, plugin.CommandBindingMissingContext)

	missingLookup := &cobra.Command{}
	missingLookup.SetContext(plugin.WithContext(context.Background(), plugin.NewAppContext(plugin.AppContextOptions{})))
	assertCommandBindingReason[T](tb, missingLookup, c.Name, plugin.CommandBindingMissingLookup)

	notFound := &cobra.Command{}
	notFound.SetContext(plugin.WithContext(context.Background(), plugin.NewAppContext(plugin.AppContextOptions{
		CommandLookup: StaticCommandLookup{},
	})))
	assertCommandBindingReason[T](tb, notFound, c.Name, plugin.CommandBindingNotFound)

	wrongPlugin := c.WrongPlugin
	if wrongPlugin == nil {
		wrongPlugin = &StubPlugin{NameVal: "wrong", DescVal: "wrong"}
	}
	wrongType := &cobra.Command{}
	wrongType.SetContext(plugin.WithContext(context.Background(), plugin.NewAppContext(plugin.AppContextOptions{
		CommandLookup: StaticCommandLookup{c.Name: wrongPlugin},
	})))
	assertCommandBindingReason[T](tb, wrongType, c.Name, plugin.CommandBindingWrongType)
}

func assertCommandBindingReason[T plugin.Plugin](tb testing.TB, cmd *cobra.Command, name string, want plugin.CommandBindingReason) {
	tb.Helper()
	_, _, err := plugin.CommandPlugin[T](cmd, name)
	if err == nil {
		tb.Fatalf("CommandPlugin(%s) error = nil", want)
	}
	var bindingErr *plugin.CommandBindingError
	if !errors.As(err, &bindingErr) {
		tb.Fatalf("CommandPlugin(%s) error type = %T, want *CommandBindingError", want, err)
	}
	if bindingErr.Reason != want {
		tb.Fatalf("CommandPlugin(%s) reason = %q, want %q", want, bindingErr.Reason, want)
	}
	if name != "" && bindingErr.Plugin != name {
		tb.Fatalf("CommandPlugin(%s) plugin = %q, want %q", want, bindingErr.Plugin, name)
	}
}

// RequireEnabledContract describes enabled/disabled fixtures for
// plugin.RequireEnabled.
type RequireEnabledContract struct {
	Enabled  interface{ IsEnabled() bool }
	Disabled interface{ IsEnabled() bool }
	Message  string
}

// AssertRequireEnabled verifies the stable DisabledPluginError boundary.
func AssertRequireEnabled(tb testing.TB, c RequireEnabledContract) {
	tb.Helper()
	message := c.Message
	if message == "" {
		message = "plugin disabled"
	}
	if err := plugin.RequireEnabled(c.Enabled, message); err != nil {
		tb.Fatalf("RequireEnabled(enabled) error = %v", err)
	}
	assertDisabledPluginError(tb, plugin.RequireEnabled(c.Disabled, message), message)
	assertDisabledPluginError(tb, plugin.RequireEnabled(nil, message), message)
}

func assertDisabledPluginError(tb testing.TB, err error, message string) {
	tb.Helper()
	if err == nil {
		tb.Fatal("RequireEnabled(disabled) error = nil")
	}
	var disabledErr *plugin.DisabledPluginError
	if !errors.As(err, &disabledErr) {
		tb.Fatalf("RequireEnabled(disabled) error type = %T, want *DisabledPluginError", err)
	}
	if err.Error() != message {
		tb.Fatalf("RequireEnabled(disabled) error = %q, want %q", err.Error(), message)
	}
}
