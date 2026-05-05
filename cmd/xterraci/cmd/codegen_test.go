package cmd

import (
	"strings"
	"testing"
)

func TestGenerateMainGo(t *testing.T) {
	src := GenerateMainGo(
		[]string{"github.com/edelwud/terraci/plugins/gitlab"},
		[]string{"github.com/example/terraci-plugin-slack"},
	)

	if !strings.Contains(src, `_ "github.com/edelwud/terraci/plugins/gitlab"`) {
		t.Error("missing builtin import")
	}
	if !strings.Contains(src, `_ "github.com/example/terraci-plugin-slack"`) {
		t.Error("missing external import")
	}
	if !strings.Contains(src, "// Built-in plugins") {
		t.Error("missing builtin section header")
	}
	if !strings.Contains(src, "// External plugins") {
		t.Error("missing external section header")
	}
	if !strings.Contains(src, "cmd.NewRootCmd") {
		t.Error("missing NewRootCmd call")
	}
	if !strings.Contains(src, `log.WithError(err).Fatal("command failed")`) {
		t.Error("missing log.Fatal call")
	}
	if strings.Contains(src, "os.Exit") {
		t.Error("should not contain os.Exit")
	}
	if !strings.Contains(src, `log "github.com/caarlos0/log"`) {
		t.Error("missing log import")
	}
	if strings.Contains(src, `"os"`) {
		t.Error("should not import os")
	}
}

func TestParsePluginSpec(t *testing.T) {
	tests := []struct {
		input   string
		module  string
		version string
		replace string
	}{
		{"github.com/foo/bar", "github.com/foo/bar", "", ""},
		{"github.com/foo/bar@v1.0.0", "github.com/foo/bar", "v1.0.0", ""},
		{"github.com/foo/bar=../local", "github.com/foo/bar", "", "../local"},
		{"github.com/foo/bar@v1.0.0=../local", "github.com/foo/bar", "v1.0.0", "../local"},
	}

	for _, tt := range tests {
		spec := parsePluginSpec(tt.input)
		if spec.Module != tt.module {
			t.Errorf("parsePluginSpec(%q).Module = %q, want %q", tt.input, spec.Module, tt.module)
		}
		if spec.Version != tt.version {
			t.Errorf("parsePluginSpec(%q).Version = %q, want %q", tt.input, spec.Version, tt.version)
		}
		if spec.Replacement != tt.replace {
			t.Errorf("parsePluginSpec(%q).Replacement = %q, want %q", tt.input, spec.Replacement, tt.replace)
		}
	}
}
