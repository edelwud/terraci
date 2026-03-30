// Package plugin provides the compile-time plugin system for TerraCi.
//
// Preferred plugin architecture for runtime-heavy built-in and external plugins:
//
//   - plugin.go: registration shell and typed BasePlugin config
//   - lifecycle.go: cheap Preflight checks only
//   - runtime.go: lazy RuntimeProvider implementation
//   - usecases.go: command orchestration over typed runtime
//   - output.go / report.go: rendering and report assembly
//
// Framework-owned lifecycle stops at configuration and preflight. Heavy clients,
// caches, and command-specific state should be created lazily from Runtime()
// and consumed by plugin commands or use-cases rather than cached during
// startup hooks.
package plugin
