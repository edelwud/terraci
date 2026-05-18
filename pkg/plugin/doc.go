// Package plugin provides the compile-time plugin system for TerraCi.
//
// # Package layout
//
// The plugin system is organized into three packages:
//
//   - pkg/plugin           — core interfaces, BasePlugin[C], AppContext, EnablePolicy, RuntimeProvider
//   - pkg/plugin/registry  — factory catalog and per-command Registry capability resolution
//   - pkg/plugin/initwiz   — init wizard types (StateMap, InitContributor, InitGroupSpec)
//
// Test helpers live in pkg/plugin/plugintest. Helpers shared between CI
// provider plugins (gitlab, github, future Bitbucket/Jenkins/Azure DevOps)
// live in plugins/internal/ciplugin. Shared report rendering for plugin-owned
// ci.Report payloads lives in plugins/internal/reportrender. These packages are
// not part of the public API.
//
// # Plugin file convention
//
// Each command-oriented plugin with a typed runtime boundary (cost, policy,
// summary, tfupdate) keeps one file per capability so the file list reads as a
// capability index:
//
//   - plugin.go       — registration shell + typed BasePlugin[C] config
//   - lifecycle.go    — cheap Preflight checks only (no network, no FS scan)
//   - runtime.go      — lazy RuntimeProvider implementation
//   - usecases.go     — command orchestration over typed runtime
//   - commands.go     — CommandProvider with cobra definitions
//   - pipeline.go     — PipelineContributor (pipeline DAG jobs)
//   - init_wizard.go  — initwiz.InitContributor (TUI form fields)
//   - output.go       — CLI rendering helpers
//   - report.go       — render-ready CI report assembly via ci.NewRenderedReport
//
// Smaller plugins (git, diskblob, inmemcache, localexec) only implement the
// capabilities they need — there is no minimum surface.
//
// # Lifecycle
//
// The framework drives every plugin through four lifecycle checkpoints per
// command run:
//
//	┌─────────────┐
//	│  Register   │  init() → registry.RegisterFactory(factory)
//	│             │  Validator.Validate() runs here — misconfigured
//	│             │  plugins panic at startup, not at first use.
//	└──────┬──────┘
//	       │
//	┌──────▼──────┐
//	│  Configure  │  ConfigLoader.DecodeAndSet — extensions.<key>
//	│             │  YAML node decoded into BasePlugin[C].cfg.
//	└──────┬──────┘
//	       │
//	┌──────▼──────┐
//	│  Preflight  │  Preflightable.Preflight(ctx, appCtx)
//	│             │  Cheap validation only. No network, no heavy state.
//	│             │  Skipped on commands with skipPreflight annotation.
//	└──────┬──────┘
//	       │
//	┌──────▼──────┐
//	│  Execute    │  RunE in command — RuntimeProvider builds heavy
//	│             │  state lazily; use-cases consume the typed runtime.
//	└─────────────┘
//
// AppContext is constructed once per command run by the framework and
// attached to cmd.Context() so plugin RunE callbacks can retrieve it via
// plugin.FromContext. It is immutable — plugins receive a snapshot of
// Config / WorkDir / ServiceDir / Resolver / pipeline contributions that does
// not change for the duration of the command.
//
// # Thread-safety contract
//
// AppContext fields are written exactly once at construction, so concurrent
// reads from any goroutine are safe without synchronization. Plugins
// should:
//
//   - Treat ctx.Config() and any field returned by accessors as read-only;
//     mutating returned pointers may surprise other plugins sharing the
//     same context.
//   - Treat ctx.Resolver() as never-nil (returns NoopResolver{} when no
//     real one is bound) and idempotent; capability lookups can run from
//     any goroutine.
//   - Treat plugin-local Config (BasePlugin[C].cfg) as command-local state
//     owned by the plugin instance.
//
// Plugin factories (the function passed to registry.RegisterFactory) MUST
// be pure: the catalog calls them once at startup for the prototype and
// once per command for the per-run plugin instance.
//
// # Capability discovery
//
// AppContext exposes typed capability resolution only. Plugins should call
// ctx.Resolver().Resolve* methods instead of enumerating or looking up
// concrete plugin names. Framework code owns raw plugin enumeration through
// registry.ByCapabilityFrom and lifecycle hooks.
//
// # Cross-plugin communication
//
// Plugins must never import each other directly. The contract surfaces are:
//
//   - capability interfaces in pkg/plugin (CI provider, change detection, …)
//   - shared types in pkg/ci (Report, ReportSection, PlanResult, …)
//   - file-based reports under appCtx.ServiceDir() ({producer}-report.json)
//   - the in-process ReportRegistry on AppContext (when transient sharing
//     within a single command run is enough)
//
// summary is the canonical consumer of file-based reports; cost/policy/
// tfupdate are the canonical producers. Producers must convert domain results
// into ci.RenderBlock values and publish reports with ci.NewRenderedReport.
// ReportSection is a value object: external plugins should not construct
// section JSON or payloads manually, and direct field access is intentionally
// unavailable. Consumers should use ci.DecodeRenderSection or
// plugins/internal/reportrender instead of importing producer-specific domain
// structs. The contract test suite for blob backends lives at
// pkg/cache/blobcache/contracttest.
package plugin
