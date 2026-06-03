package plugin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// CommandProvider adds CLI subcommands to TerraCi. The framework calls
// CommandSpecs() once during root command registration. Inside RunE, plugins
// should use CommandPlugin[T] to resolve the per-run AppContext plus the
// command-scoped plugin instance in one typed call.
type CommandProvider interface {
	Plugin
	CommandSpecs() ([]CommandSpec, error)
}

// CommandRegistrationError wraps command contribution failures with plugin and
// command context.
type CommandRegistrationError struct {
	Plugin  string
	Command string
	Err     error
}

func (e CommandRegistrationError) Error() string {
	if e.Err == nil {
		return "register plugin command"
	}
	if e.Plugin == "" && e.Command == "" {
		return fmt.Sprintf("register plugin command: %v", e.Err)
	}
	if e.Command == "" {
		return fmt.Sprintf("register plugin commands for %s: %v", e.Plugin, e.Err)
	}
	if e.Plugin == "" {
		return fmt.Sprintf("register plugin command %s: %v", e.Command, e.Err)
	}
	return fmt.Sprintf("register plugin command %s for %s: %v", e.Command, e.Plugin, e.Err)
}

func (e CommandRegistrationError) Unwrap() error { return e.Err }

// CommandRunFunc is the command callback used by a CommandSpec.
type CommandRunFunc func(*cobra.Command, []string) error

// CommandConfigureFunc configures Cobra-only presentation details such as
// flags. It must not capture command-scoped plugin instances.
type CommandConfigureFunc func(*cobra.Command) error

// CommandSpecOptions describes one plugin command contribution.
type CommandSpecOptions struct {
	Use         string
	Short       string
	Long        string
	Example     string
	Aliases     []string
	Args        cobra.PositionalArgs
	Configure   CommandConfigureFunc
	RunE        CommandRunFunc
	Subcommands []CommandSpec
}

// CommandSpec is a validated plugin command contribution. The framework owns
// conversion to mutable Cobra commands.
type CommandSpec struct {
	use         string
	short       string
	long        string
	example     string
	aliases     []string
	args        cobra.PositionalArgs
	configure   CommandConfigureFunc
	runE        CommandRunFunc
	subcommands []CommandSpec
	constructed bool
}

// NewCommandSpec validates and returns one plugin command contribution.
func NewCommandSpec(opts CommandSpecOptions) (CommandSpec, error) {
	spec := CommandSpec{
		use:         opts.Use,
		short:       opts.Short,
		long:        opts.Long,
		example:     opts.Example,
		aliases:     append([]string(nil), opts.Aliases...),
		args:        opts.Args,
		configure:   opts.Configure,
		runE:        opts.RunE,
		subcommands: append([]CommandSpec(nil), opts.Subcommands...),
		constructed: true,
	}
	if err := validateCommandSpec(spec); err != nil {
		return CommandSpec{}, err
	}
	return spec, nil
}

// Use returns the Cobra use line.
func (s CommandSpec) Use() string { return s.use }

// Short returns the short help text.
func (s CommandSpec) Short() string { return s.short }

// Long returns the long help text.
func (s CommandSpec) Long() string { return s.long }

// Example returns command examples.
func (s CommandSpec) Example() string { return s.example }

// Aliases returns defensive alias copies.
func (s CommandSpec) Aliases() []string {
	if len(s.aliases) == 0 {
		return nil
	}
	return append([]string(nil), s.aliases...)
}

// Subcommands returns defensive subcommand copies.
func (s CommandSpec) Subcommands() []CommandSpec {
	if len(s.subcommands) == 0 {
		return nil
	}
	return append([]CommandSpec(nil), s.subcommands...)
}

// HasRun reports whether the command has a RunE callback.
func (s CommandSpec) HasRun() bool { return s.runE != nil }

// BuildCommand builds a mutable Cobra command from a validated command spec.
func BuildCommand(spec CommandSpec) (*cobra.Command, error) {
	if err := validateCommandSpec(spec); err != nil {
		return nil, err
	}

	cmd := &cobra.Command{
		Use:     spec.use,
		Short:   spec.short,
		Long:    spec.long,
		Example: spec.example,
		Aliases: spec.Aliases(),
		Args:    spec.args,
		RunE:    spec.runE,
	}
	if spec.configure != nil {
		if err := spec.configure(cmd); err != nil {
			return nil, fmt.Errorf("configure command %q: %w", commandSpecName(spec), err)
		}
	}
	for i := range spec.subcommands {
		childSpec := spec.subcommands[i]
		child, err := BuildCommand(childSpec)
		if err != nil {
			return nil, err
		}
		cmd.AddCommand(child)
	}
	return cmd, nil
}

func validateCommandSpec(spec CommandSpec) error {
	if !spec.constructed {
		return errors.New("command spec must be built with NewCommandSpec")
	}
	if strings.TrimSpace(spec.use) == "" {
		return errors.New("command use is required")
	}
	if len(spec.subcommands) == 0 && spec.runE == nil {
		return fmt.Errorf("command %q must define RunE or subcommands", commandSpecName(spec))
	}
	if err := validateCommandAliases(commandSpecName(spec), spec.aliases); err != nil {
		return err
	}
	if err := validateSiblingCommandSpecs(spec.subcommands); err != nil {
		return err
	}
	for i := range spec.subcommands {
		child := spec.subcommands[i]
		if err := validateCommandSpec(child); err != nil {
			return err
		}
	}
	return nil
}

func validateSiblingCommandSpecs(specs []CommandSpec) error {
	seen := make(map[string]string)
	for i := range specs {
		spec := specs[i]
		name := commandSpecName(spec)
		if name == "" {
			continue
		}
		if previous, exists := seen[name]; exists {
			return fmt.Errorf("duplicate command name %q in siblings %q and %q", name, previous, name)
		}
		seen[name] = name
		for _, alias := range spec.aliases {
			if previous, exists := seen[alias]; exists {
				return fmt.Errorf("duplicate command alias %q conflicts with %q", alias, previous)
			}
			seen[alias] = name
		}
	}
	return nil
}

func validateCommandAliases(command string, aliases []string) error {
	seen := make(map[string]struct{})
	for _, alias := range aliases {
		if strings.TrimSpace(alias) == "" {
			return fmt.Errorf("command %q has empty alias", command)
		}
		if _, exists := seen[alias]; exists {
			return fmt.Errorf("command %q has duplicate alias %q", command, alias)
		}
		seen[alias] = struct{}{}
	}
	return nil
}

func commandSpecName(spec CommandSpec) string {
	parts := strings.Fields(spec.use)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// VersionProvider plugins contribute version info to `terraci version`.
type VersionProvider interface {
	Plugin
	VersionInfo() map[string]string
}
