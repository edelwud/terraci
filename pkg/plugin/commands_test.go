package plugin

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewCommandSpecBuildsCobraCommand(t *testing.T) {
	t.Parallel()

	called := false
	spec, err := NewCommandSpec(CommandSpecOptions{
		Use:     "hello",
		Short:   "short",
		Long:    "long",
		Example: "terraci hello",
		Aliases: []string{"hi"},
		Args:    cobra.NoArgs,
		Configure: func(cmd *cobra.Command) error {
			cmd.Flags().String("format", "text", "output format")
			return nil
		},
		RunE: func(*cobra.Command, []string) error {
			called = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewCommandSpec() error = %v", err)
	}

	aliases := spec.Aliases()
	aliases[0] = "mutated"
	if got := spec.Aliases()[0]; got != "hi" {
		t.Fatalf("Aliases() leaked mutation: %q", got)
	}

	cmd, err := BuildCommand(spec)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}
	if cmd.Use != "hello" || cmd.Short != "short" || cmd.Long != "long" || cmd.Example != "terraci hello" {
		t.Fatalf("unexpected command presentation: %#v", cmd)
	}
	if got := cmd.Aliases[0]; got != "hi" {
		t.Fatalf("command alias = %q, want hi", got)
	}
	if cmd.Flags().Lookup("format") == nil {
		t.Fatal("BuildCommand() did not run Configure")
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if !called {
		t.Fatal("RunE was not called")
	}
}

func TestNewCommandSpecValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts CommandSpecOptions
		want string
	}{
		{
			name: "empty use",
			opts: CommandSpecOptions{RunE: func(*cobra.Command, []string) error { return nil }},
			want: "command use is required",
		},
		{
			name: "leaf without run",
			opts: CommandSpecOptions{Use: "hello"},
			want: "must define RunE or subcommands",
		},
		{
			name: "empty alias",
			opts: CommandSpecOptions{Use: "hello", Aliases: []string{""}, RunE: func(*cobra.Command, []string) error { return nil }},
			want: "empty alias",
		},
		{
			name: "duplicate alias",
			opts: CommandSpecOptions{Use: "hello", Aliases: []string{"hi", "hi"}, RunE: func(*cobra.Command, []string) error { return nil }},
			want: "duplicate alias",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewCommandSpec(tt.opts)
			if err == nil {
				t.Fatal("NewCommandSpec() error = nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestNewCommandSpecValidatesDuplicateSubcommands(t *testing.T) {
	t.Parallel()

	first, err := NewCommandSpec(CommandSpecOptions{Use: "check", RunE: func(*cobra.Command, []string) error { return nil }})
	if err != nil {
		t.Fatalf("NewCommandSpec(first) error = %v", err)
	}
	second, err := NewCommandSpec(CommandSpecOptions{Use: "pull", Aliases: []string{"check"}, RunE: func(*cobra.Command, []string) error { return nil }})
	if err != nil {
		t.Fatalf("NewCommandSpec(second) error = %v", err)
	}

	_, err = NewCommandSpec(CommandSpecOptions{Use: "policy", Subcommands: []CommandSpec{first, second}})
	if err == nil {
		t.Fatal("NewCommandSpec(parent) error = nil")
	}
	if !strings.Contains(err.Error(), "duplicate command alias") {
		t.Fatalf("error = %q, want duplicate alias", err.Error())
	}
}

func TestBuildCommandRejectsZeroSpecAndConfigureError(t *testing.T) {
	t.Parallel()

	if _, err := BuildCommand(CommandSpec{}); err == nil || !strings.Contains(err.Error(), "NewCommandSpec") {
		t.Fatalf("BuildCommand(zero) error = %v, want constructor hint", err)
	}

	configErr := errors.New("bad flags")
	spec, err := NewCommandSpec(CommandSpecOptions{
		Use:       "hello",
		Configure: func(*cobra.Command) error { return configErr },
		RunE:      func(*cobra.Command, []string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewCommandSpec() error = %v", err)
	}
	_, err = BuildCommand(spec)
	if !errors.Is(err, configErr) {
		t.Fatalf("BuildCommand() error = %v, want wrapped config error", err)
	}
}
