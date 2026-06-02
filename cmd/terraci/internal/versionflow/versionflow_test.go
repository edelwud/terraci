package versionflow

import (
	"bytes"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
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

func TestBuildAndWrite(t *testing.T) {
	t.Parallel()

	source := registry.NewFromFactories(func() plugin.Plugin {
		return testPlugin{name: "policy", info: map[string]string{"opa": "1.0"}}
	})
	result := Build(Metadata{Version: "v1", Commit: "abc", Date: "today"}, source)

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
