package test

import (
	"strings"
	"testing"
)

// Smoke tests for `terraci --help` and per-subcommand help. These catch
// regressions in command wiring (e.g. a subcommand registered for the wrong
// parent, missing flags) without exercising the actual command logic.

func TestHelp_RootListsCoreCommands(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "--help")
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}

	// Each core command must appear in root help output.
	for _, name := range []string{
		"generate", "graph", "validate", "init", "version", "schema",
	} {
		if !strings.Contains(output, name) {
			t.Errorf("root --help missing core command %q\noutput:\n%s", name, output)
		}
	}
}

func TestHelp_RootListsPluginCommands(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "--help")
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}

	// Plugin-provided commands must appear too — registered through
	// CommandProvider at root construction time.
	for _, name := range []string{"cost", "policy", "summary", "tfupdate", "local-exec"} {
		if !strings.Contains(output, name) {
			t.Errorf("root --help missing plugin command %q", name)
		}
	}
}

func TestHelp_GenerateMentionsKeyFlags(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "generate", "--help")
	if err != nil {
		t.Fatalf("generate --help failed: %v", err)
	}

	// Critical flags users rely on for CI integration.
	for _, flag := range []string{"--output", "--changed-only", "--base-ref", "--dry-run", "--plan-only", "--format"} {
		if !strings.Contains(output, flag) {
			t.Errorf("generate --help missing flag %q\noutput:\n%s", flag, output)
		}
	}
}

func TestHelp_InitMentionsKeyFlags(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "init", "--help")
	if err != nil {
		t.Fatalf("init --help failed: %v", err)
	}

	for _, flag := range []string{"--ci", "--provider", "--binary", "--pattern", "--force"} {
		if !strings.Contains(output, flag) {
			t.Errorf("init --help missing flag %q", flag)
		}
	}

	// --image is the flag that R1.7 explicitly removed; lock it out so a
	// regression doesn't accidentally re-add it.
	if strings.Contains(output, "--image") {
		t.Error("init --help should NOT advertise --image (removed in R1)")
	}
}

func TestHelp_UnknownSubcommandFails(t *testing.T) {
	dir := fixtureDir(t, "basic")

	err := runTerraCi(t, dir, "this-command-does-not-exist")
	if err == nil {
		t.Fatal("unknown subcommand should return error")
	}
}
