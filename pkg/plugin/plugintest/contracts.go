package plugintest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

// BaseConfigPlugin is the public surface implemented by plugin.BasePlugin[C].
// Contract tests use this interface instead of depending on the concrete
// embedded type, so third-party plugins can wrap BasePlugin in their own
// plugin struct and still reuse the assertions.
type BaseConfigPlugin[C plugin.ConfigCloner[C]] interface {
	plugin.ConfigLoader
	Config() C
	SetTypedConfig(C)
}

// BaseConfigPluginContract describes the fixtures needed to verify the
// immutable-config contract for a BasePlugin-backed plugin.
type BaseConfigPluginContract[C plugin.ConfigCloner[C]] struct {
	Plugin     BaseConfigPlugin[C]
	Default    C
	Configured C
	Decoded    C
	Mutate     func(C)
	Equal      func(got, want C) bool
}

// AssertBaseConfigPlugin verifies the canonical config behavior external
// plugin authors rely on: configs are Clone()able, NewConfig returns a fresh
// default, and Config/SetTypedConfig/DecodeAndSet do not leak mutable state.
func AssertBaseConfigPlugin[C plugin.ConfigCloner[C]](tb testing.TB, c BaseConfigPluginContract[C]) {
	tb.Helper()
	if c.Plugin == nil {
		tb.Fatal("Plugin is nil")
	}
	if c.Mutate == nil {
		tb.Fatal("Mutate is nil")
	}
	if c.Equal == nil {
		tb.Fatal("Equal is nil")
	}

	assertConfigEqual(tb, "NewConfig()", asConfig[C](tb, c.Plugin.NewConfig()), c.Default, c.Equal)
	firstDefault := asConfig[C](tb, c.Plugin.NewConfig())
	c.Mutate(firstDefault)
	assertConfigEqual(tb, "NewConfig() after mutating prior default", asConfig[C](tb, c.Plugin.NewConfig()), c.Default, c.Equal)

	configuredWant := c.Configured.Clone()
	c.Plugin.SetTypedConfig(c.Configured)
	c.Mutate(c.Configured)
	assertConfigEqual(tb, "Config() after SetTypedConfig", c.Plugin.Config(), configuredWant, c.Equal)
	gotConfigured := c.Plugin.Config()
	c.Mutate(gotConfigured)
	assertConfigEqual(tb, "Config() after mutating returned config", c.Plugin.Config(), configuredWant, c.Equal)

	decodedWant := c.Decoded.Clone()
	if err := c.Plugin.DecodeAndSet(func(target any) error {
		ptr, ok := target.(*C)
		if !ok {
			tb.Fatalf("DecodeAndSet target type = %T, want *config", target)
		}
		*ptr = c.Decoded
		return nil
	}); err != nil {
		tb.Fatalf("DecodeAndSet() error = %v", err)
	}
	c.Mutate(c.Decoded)
	assertConfigEqual(tb, "Config() after DecodeAndSet", c.Plugin.Config(), decodedWant, c.Equal)
	gotDecoded := c.Plugin.Config()
	c.Mutate(gotDecoded)
	assertConfigEqual(tb, "Config() after mutating decoded return value", c.Plugin.Config(), decodedWant, c.Equal)
}

func asConfig[C plugin.ConfigCloner[C]](tb testing.TB, value any) C {
	tb.Helper()
	cfg, ok := value.(C)
	if !ok {
		var zero C
		tb.Fatalf("config type = %T, want %T", value, zero)
	}
	return cfg
}

func assertConfigEqual[C plugin.ConfigCloner[C]](tb testing.TB, label string, got, want C, equal func(C, C) bool) {
	tb.Helper()
	if !equal(got, want) {
		tb.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}

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

// PipelineContributorContract describes expected generic contribution shape.
type PipelineContributorContract struct {
	Contributor      plugin.PipelineContributor
	AppContext       *plugin.AppContext
	ExpectedJobNames []string
}

// AssertPipelineContributor verifies deterministic contribution shape without
// asserting plugin-specific command/resource details.
func AssertPipelineContributor(tb testing.TB, c PipelineContributorContract) {
	tb.Helper()
	if c.Contributor == nil {
		tb.Fatal("Contributor is nil")
	}
	appCtx := c.AppContext
	if appCtx == nil {
		appCtx = plugin.NewAppContext(plugin.AppContextOptions{})
	}

	first := c.Contributor.PipelineContribution(appCtx)
	if first == nil {
		tb.Fatal("PipelineContribution() = nil")
	}
	gotNames := contributedJobNames(first)
	if len(c.ExpectedJobNames) > 0 && !slices.Equal(gotNames, c.ExpectedJobNames) {
		tb.Fatalf("contributed job names = %v, want %v", gotNames, c.ExpectedJobNames)
	}
	for _, name := range gotNames {
		if name == "" {
			tb.Fatalf("contributed job names contain an empty name: %v", gotNames)
		}
	}

	second := c.Contributor.PipelineContribution(appCtx)
	if second == nil {
		tb.Fatal("second PipelineContribution() = nil")
	}
	if gotAgain := contributedJobNames(second); !slices.Equal(gotAgain, gotNames) {
		tb.Fatalf("PipelineContribution() job names are not deterministic: first %v, second %v", gotNames, gotAgain)
	}
}

func contributedJobNames(contribution *pipeline.Contribution) []string {
	if contribution == nil || len(contribution.Jobs) == 0 {
		return nil
	}
	names := make([]string, 0, len(contribution.Jobs))
	for _, job := range contribution.Jobs {
		names = append(names, job.Name)
	}
	return names
}
