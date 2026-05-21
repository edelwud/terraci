package versionflow

import (
	"bytes"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
)

type testPlugin struct {
	name string
	info map[string]string
}

func (p testPlugin) Name() string        { return p.name }
func (p testPlugin) Description() string { return p.name + " plugin" }
func (p testPlugin) VersionInfo() map[string]string {
	return p.info
}

type testSource []plugin.Plugin

func (s testSource) All() []plugin.Plugin { return []plugin.Plugin(s) }
func (s testSource) VersionProviders() []plugin.VersionProvider {
	var providers []plugin.VersionProvider
	for _, p := range s {
		if provider, ok := p.(plugin.VersionProvider); ok {
			providers = append(providers, provider)
		}
	}
	return providers
}

func TestBuildAndWrite(t *testing.T) {
	t.Parallel()

	result := Build(Metadata{Version: "v1", Commit: "abc", Date: "today"}, testSource{
		testPlugin{name: "policy", info: map[string]string{"opa": "1.0"}},
	})

	var buf bytes.Buffer
	if err := Write(&buf, result); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	output := buf.String()
	for _, want := range []string{
		"terraci v1",
		"  commit: abc",
		"  built:  today",
		"  opa: 1.0",
		"    - policy: policy plugin",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want %q", output, want)
		}
	}
}
