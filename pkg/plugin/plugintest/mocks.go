package plugintest

import (
	"context"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/plugin"
)

// StubPlugin is a minimal Plugin implementation for testing.
type StubPlugin struct {
	NameVal string
	DescVal string
}

func (p *StubPlugin) Name() string        { return p.NameVal }
func (p *StubPlugin) Description() string { return p.DescVal }

// StubChangeDetector implements ChangeDetectionProvider for testing.
type StubChangeDetector struct {
	StubPlugin
	ChangedModules   []*discovery.Module
	ChangedFiles     []string
	ChangedLibraries []string
	Err              error
}

func (d *StubChangeDetector) DetectChangedModules(_ context.Context, _ *plugin.AppContext, _ string, _ *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	return d.ChangedModules, d.ChangedFiles, d.Err
}

func (d *StubChangeDetector) DetectChangedLibraries(_ context.Context, _ *plugin.AppContext, _ string, _ []string) ([]string, error) {
	return d.ChangedLibraries, d.Err
}

// StubConfigPlugin embeds BasePlugin for testing config-aware scenarios.
type StubConfigPlugin struct {
	plugin.BasePlugin[*StubConfig]
}

// StubConfig is a minimal config struct for testing.
type StubConfig struct {
	Enabled bool
}
