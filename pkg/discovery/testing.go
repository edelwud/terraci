package discovery

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/config"
)

// TestModule creates a Module with the default 4-segment pattern for testing.
func TestModule(service, env, region, module string) *Module {
	values := []string{service, env, region, module}
	relPath := filepath.Join(service, env, region, module)
	return NewModule(config.DefaultSegments(), values, relPath, relPath)
}

// TestLibraryModule creates a library Module rooted at the given relative path
// (e.g. "_modules/kafka") for use in tests that need IsLibrary=true modules.
// Path equals RelativePath; pass an absolute path explicitly via the optional
// abs argument when test logic expects Module.Path to differ from RelativePath
// (graph rendering matches libraryUsage by absolute path).
func TestLibraryModule(relPath string, abs ...string) *Module {
	mod := NewModule(config.DefaultSegments(), nil, relPath, relPath)
	mod.IsLibrary = true
	if len(abs) > 0 && abs[0] != "" {
		mod.Path = abs[0]
	}
	return mod
}
