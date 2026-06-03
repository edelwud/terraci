package plugintest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/plugin"
)

// StaticCommandSource is a minimal command-scoped plugin source for
// CommandPlugin contract tests.
type StaticCommandSource map[string]plugin.Plugin

// LookupCommandPlugin implements plugin.CommandBindingSource.
func (l StaticCommandSource) LookupCommandPlugin(name string) (plugin.Plugin, bool) {
	p, ok := l[name]
	return p, ok
}

// BindCommandContext binds appCtx and source into parent for CommandPlugin
// tests.
func BindCommandContext(parent context.Context, tb testing.TB, appCtx *plugin.AppContext, source plugin.CommandBindingSource) context.Context {
	tb.Helper()
	binding, err := plugin.NewCommandBinding(plugin.CommandBindingOptions{
		AppContext: appCtx,
		Source:     source,
	})
	if err != nil {
		tb.Fatalf("NewCommandBinding() error = %v", err)
	}
	return plugin.BindCommandContext(parent, binding)
}

// BindCommandPlugin binds a single command-scoped plugin into parent.
func BindCommandPlugin(parent context.Context, tb testing.TB, appCtx *plugin.AppContext, name string, p plugin.Plugin) context.Context {
	tb.Helper()
	return BindCommandContext(parent, tb, appCtx, StaticCommandSource{name: p})
}

// CommandBindingContract describes the fixtures for CommandPlugin[T].
type CommandBindingContract[T plugin.Plugin] struct {
	Name           string
	Plugin         T
	WrongPlugin    plugin.Plugin
	AssertResolved func(testing.TB, T)
}

// CommandProviderContract describes the fixtures for a command provider.
type CommandProviderContract struct {
	Provider      plugin.CommandProvider
	ExpectedUses  []string
	AssertCommand func(testing.TB, []*cobra.Command)
}

// AssertCommandProvider verifies command specs build into Cobra commands with
// deterministic use lines and optional provider-specific flag/help assertions.
func AssertCommandProvider(tb testing.TB, c CommandProviderContract) {
	tb.Helper()
	if c.Provider == nil {
		tb.Fatal("Provider is nil")
	}
	specs, err := c.Provider.CommandSpecs()
	if err != nil {
		tb.Fatalf("CommandSpecs() error = %v", err)
	}
	commands := make([]*cobra.Command, 0, len(specs))
	for i := range specs {
		spec := specs[i]
		cmd, err := plugin.BuildCommand(spec)
		if err != nil {
			tb.Fatalf("BuildCommand(%q) error = %v", spec.Use(), err)
		}
		commands = append(commands, cmd)
	}
	if len(c.ExpectedUses) > 0 {
		got := make([]string, 0, len(commands))
		for _, cmd := range commands {
			got = append(got, cmd.Use)
		}
		if !slices.Equal(got, c.ExpectedUses) {
			tb.Fatalf("command uses = %v, want %v", got, c.ExpectedUses)
		}
	}
	if c.AssertCommand != nil {
		c.AssertCommand(tb, commands)
	}
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

	appCtx := plugin.NewAppContext(plugin.AppContextOptions{})
	binding, err := plugin.NewCommandBinding(plugin.CommandBindingOptions{
		AppContext: appCtx,
		Source:     StaticCommandSource{c.Name: c.Plugin},
	})
	if err != nil {
		tb.Fatalf("NewCommandBinding(success) error = %v", err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(plugin.BindCommandContext(context.Background(), binding))
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

	if _, err := plugin.NewCommandBinding(plugin.CommandBindingOptions{AppContext: plugin.NewAppContext(plugin.AppContextOptions{})}); err == nil {
		tb.Fatal("NewCommandBinding(missing lookup) error = nil")
	} else {
		assertBindingError(tb, err, plugin.CommandBindingMissingLookup, "")
	}

	notFound := &cobra.Command{}
	notFound.SetContext(bindTestCommand(tb, plugin.NewAppContext(plugin.AppContextOptions{}), StaticCommandSource{}))
	assertCommandBindingReason[T](tb, notFound, c.Name, plugin.CommandBindingNotFound)

	wrongPlugin := c.WrongPlugin
	if wrongPlugin == nil {
		wrongPlugin = &StubPlugin{NameVal: "wrong", DescVal: "wrong"}
	}
	wrongType := &cobra.Command{}
	wrongType.SetContext(bindTestCommand(tb, plugin.NewAppContext(plugin.AppContextOptions{}), StaticCommandSource{c.Name: wrongPlugin}))
	assertCommandBindingReason[T](tb, wrongType, c.Name, plugin.CommandBindingWrongType)
}

func bindTestCommand(tb testing.TB, appCtx *plugin.AppContext, source plugin.CommandBindingSource) context.Context {
	tb.Helper()
	binding, err := plugin.NewCommandBinding(plugin.CommandBindingOptions{
		AppContext: appCtx,
		Source:     source,
	})
	if err != nil {
		tb.Fatalf("NewCommandBinding() error = %v", err)
	}
	return plugin.BindCommandContext(context.Background(), binding)
}

func assertCommandBindingReason[T plugin.Plugin](tb testing.TB, cmd *cobra.Command, name string, want plugin.CommandBindingReason) {
	tb.Helper()
	_, _, err := plugin.CommandPlugin[T](cmd, name)
	if err == nil {
		tb.Fatalf("CommandPlugin(%s) error = nil", want)
	}
	assertBindingError(tb, err, want, name)
}

func assertBindingError(tb testing.TB, err error, want plugin.CommandBindingReason, name string) {
	tb.Helper()
	var bindingErr *plugin.CommandBindingError
	if !errors.As(err, &bindingErr) {
		tb.Fatalf("command binding error type = %T, want *CommandBindingError", err)
	}
	if bindingErr.Reason != want {
		tb.Fatalf("CommandBindingError reason = %q, want %q", bindingErr.Reason, want)
	}
	if name != "" && bindingErr.Plugin != name {
		tb.Fatalf("CommandBindingError plugin = %q, want %q", bindingErr.Plugin, name)
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
