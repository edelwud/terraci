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
