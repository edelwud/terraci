package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/pipeline"
)

// CommandBindingReason identifies a command binding failure class.
type CommandBindingReason string

const (
	CommandBindingNilCommand     CommandBindingReason = "nil_command"
	CommandBindingMissingContext CommandBindingReason = "missing_context"
	CommandBindingMissingLookup  CommandBindingReason = "missing_lookup"
	CommandBindingNotFound       CommandBindingReason = "not_found"
	CommandBindingWrongType      CommandBindingReason = "wrong_type"
)

// CommandBindingSource resolves command-scoped plugin instances for
// CommandPlugin. It is framework-owned; plugin authors should not implement it
// unless they are building tests or custom command wiring.
type CommandBindingSource interface {
	LookupCommandPlugin(name string) (Plugin, bool)
}

// CommandBindingOptions describes a command binding value.
type CommandBindingOptions struct {
	AppContext            *AppContext
	Source                CommandBindingSource
	PipelineContributions []*pipeline.Contribution
}

// CommandContext is the command-scoped SDK value exposed to command callbacks.
// It carries the runtime AppContext plus framework planning state collected for
// this command run.
type CommandContext struct {
	appCtx        *AppContext
	contributions []*pipeline.Contribution
}

// AppContext returns the runtime plugin context for this command run.
func (c CommandContext) AppContext() *AppContext { return c.appCtx }

// PipelineContributions returns enabled pipeline contributions collected by
// runflow for this command run.
func (c CommandContext) PipelineContributions() []*pipeline.Contribution {
	return clonePipelineContributions(c.contributions)
}

// CommandBinding is the command-scoped bridge between a cobra callback, its
// immutable AppContext, and the plugin instance created for that command run.
type CommandBinding struct {
	commandCtx CommandContext
	source     CommandBindingSource
}

// NewCommandBinding validates and builds a command binding value.
func NewCommandBinding(opts CommandBindingOptions) (*CommandBinding, error) {
	if opts.AppContext == nil {
		return nil, &CommandBindingError{Reason: CommandBindingMissingContext}
	}
	if opts.Source == nil {
		return nil, &CommandBindingError{Reason: CommandBindingMissingLookup}
	}
	return &CommandBinding{
		commandCtx: CommandContext{
			appCtx:        opts.AppContext,
			contributions: clonePipelineContributions(opts.PipelineContributions),
		},
		source: opts.Source,
	}, nil
}

// AppContext returns the immutable command AppContext.
func (b *CommandBinding) AppContext() *AppContext {
	if b == nil {
		return nil
	}
	return b.commandCtx.AppContext()
}

// CommandContext returns the command-scoped SDK context.
func (b *CommandBinding) CommandContext() CommandContext {
	if b == nil {
		return CommandContext{}
	}
	return CommandContext{
		appCtx:        b.commandCtx.appCtx,
		contributions: clonePipelineContributions(b.commandCtx.contributions),
	}
}

type commandBindingKey struct{}

// BindCommandContext returns a child context carrying the framework command
// binding for CommandPlugin.
func BindCommandContext(parent context.Context, binding *CommandBinding) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, commandBindingKey{}, binding)
}

func commandBindingFromContext(ctx context.Context) *CommandBinding {
	if ctx == nil {
		return nil
	}
	binding, ok := ctx.Value(commandBindingKey{}).(*CommandBinding)
	if !ok {
		return nil
	}
	return binding
}

// CommandBindingError is returned when a cobra callback cannot be bound to the
// current command-scoped AppContext or plugin instance.
type CommandBindingError struct {
	Plugin       string
	Reason       CommandBindingReason
	ActualType   string
	ExpectedType string
}

func (e *CommandBindingError) Error() string {
	if e == nil {
		return "command binding error"
	}
	subject := "command context"
	if e.Plugin != "" {
		subject = fmt.Sprintf("command plugin %q", e.Plugin)
	}
	switch e.Reason {
	case CommandBindingNilCommand:
		return subject + ": cobra command is nil"
	case CommandBindingMissingContext:
		return subject + ": plugin context is not bound"
	case CommandBindingMissingLookup:
		return subject + ": command lookup is not bound"
	case CommandBindingNotFound:
		return subject + ": command-scoped instance not found"
	case CommandBindingWrongType:
		return fmt.Sprintf("%s: command-scoped instance has type %s (want %s)", subject, e.ActualType, e.ExpectedType)
	default:
		return subject + ": command binding failed"
	}
}

// DisabledPluginError is returned when a command targets a disabled plugin.
type DisabledPluginError struct {
	Message string
}

func (e *DisabledPluginError) Error() string {
	if e == nil || e.Message == "" {
		return "plugin is not enabled"
	}
	return e.Message
}

// commandInstance returns the command-scoped plugin instance matching name.
func commandInstance[T Plugin](binding *CommandBinding, name string) (T, error) {
	var zero T
	if binding == nil || binding.commandCtx.appCtx == nil {
		return zero, &CommandBindingError{Plugin: name, Reason: CommandBindingMissingContext}
	}
	if binding.source == nil {
		return zero, &CommandBindingError{Plugin: name, Reason: CommandBindingMissingLookup}
	}
	current, ok := binding.source.LookupCommandPlugin(name)
	if !ok {
		return zero, &CommandBindingError{Plugin: name, Reason: CommandBindingNotFound}
	}
	typed, ok := current.(T)
	if !ok {
		return zero, &CommandBindingError{
			Plugin:       name,
			Reason:       CommandBindingWrongType,
			ActualType:   fmt.Sprintf("%T", current),
			ExpectedType: fmt.Sprintf("%T", zero),
		}
	}
	return typed, nil
}

// CommandPlugin resolves the command-scoped AppContext and plugin instance for
// a cobra callback. It is the preferred command boundary for built-in and
// external plugins.
func CommandPlugin[T Plugin](cmd *cobra.Command, name string) (CommandContext, T, error) {
	var zero T
	if cmd == nil {
		return CommandContext{}, zero, &CommandBindingError{Plugin: name, Reason: CommandBindingNilCommand}
	}
	binding := commandBindingFromContext(cmd.Context())
	if binding == nil || binding.commandCtx.appCtx == nil {
		return CommandContext{}, zero, &CommandBindingError{Plugin: name, Reason: CommandBindingMissingContext}
	}
	current, err := commandInstance[T](binding, name)
	if err != nil {
		var bindingErr *CommandBindingError
		if errors.As(err, &bindingErr) {
			bindingErr.Plugin = name
		}
		return binding.CommandContext(), zero, err
	}
	return binding.CommandContext(), current, nil
}

// RequireEnabled returns message when p reports disabled.
func RequireEnabled(p interface{ IsEnabled() bool }, message string) error {
	if message == "" {
		message = "plugin is not enabled"
	}
	if p == nil {
		return &DisabledPluginError{Message: message}
	}
	if !p.IsEnabled() {
		return &DisabledPluginError{Message: message}
	}
	return nil
}

func clonePipelineContributions(contribs []*pipeline.Contribution) []*pipeline.Contribution {
	if len(contribs) == 0 {
		return nil
	}
	clone := make([]*pipeline.Contribution, len(contribs))
	for i, contribution := range contribs {
		clone[i] = contribution.Clone()
	}
	return clone
}
