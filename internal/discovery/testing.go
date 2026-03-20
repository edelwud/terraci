package discovery

import "path/filepath"

// TestModule creates a Module with the default 4-segment pattern for testing.
func TestModule(service, env, region, module string) *Module {
	segments := []string{"service", "environment", "region", "module"}
	values := []string{service, env, region, module}
	relPath := filepath.Join(service, env, region, module)
	return NewModule(segments, values, relPath, relPath)
}
