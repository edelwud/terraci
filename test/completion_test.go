package test

import (
	"testing"
)

func TestCompletion_Bash(t *testing.T) {
	output, err := captureTerraCi(t, t.TempDir(), "completion", "bash")
	if err != nil {
		t.Fatalf("completion bash failed: %v", err)
	}
	assertContains(t, output, "complete")
}

func TestCompletion_Zsh(t *testing.T) {
	output, err := captureTerraCi(t, t.TempDir(), "completion", "zsh")
	if err != nil {
		t.Fatalf("completion zsh failed: %v", err)
	}
	assertContains(t, output, "compdef")
}

func TestCompletion_Fish(t *testing.T) {
	output, err := captureTerraCi(t, t.TempDir(), "completion", "fish")
	if err != nil {
		t.Fatalf("completion fish failed: %v", err)
	}
	assertContains(t, output, "complete -c terraci")
}

func TestCompletion_PowerShell(t *testing.T) {
	output, err := captureTerraCi(t, t.TempDir(), "completion", "powershell")
	if err != nil {
		t.Fatalf("completion powershell failed: %v", err)
	}
	assertContains(t, output, "Register-ArgumentCompleter")
}

func TestCompletion_NoArgument(t *testing.T) {
	err := runTerraCi(t, t.TempDir(), "completion")
	if err == nil {
		t.Fatal("expected error when no shell argument provided")
	}
}

func TestCompletion_InvalidShell(t *testing.T) {
	err := runTerraCi(t, t.TempDir(), "completion", "perl")
	if err == nil {
		t.Fatal("expected error for invalid shell argument")
	}
}
