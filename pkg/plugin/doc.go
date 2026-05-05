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
// live in plugins/internal/ciplugin and are not part of the public API.
//
// # Plugin file convention
//
// Each runtime-heavy plugin (cost, policy, tfupdate) keeps one file per
// capability so the file list reads as a capability index:
//
//   - plugin.go       — registration shell + typed BasePlugin[C] config
//   - lifecycle.go    — cheap Preflight checks only (no network, no FS scan)
//   - runtime.go      — lazy RuntimeProvider implementation
//   - usecases.go     — command orchestration over typed runtime
//   - commands.go     — CommandProvider with cobra definitions
//   - pipeline.go     — PipelineContributor (steps + standalone jobs)
//   - init_wizard.go  — initwiz.InitContributor (TUI form fields)
//   - output.go       — CLI rendering helpers
//   - report.go       — typed CI report assembly via ci.EncodeSection
//
// Smaller plugins (git, diskblob, inmemcache, summary, localexec) only
// implement the capabilities they need — there is no minimum surface.
//
// # Lifecycle
//
// The framework drives every plugin through five stages:
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
//	│   Freeze    │  AppContext.Freeze() — core fields read-only.
//	│             │  Plugin-local config remains mutable for FlagOverridable.
//	└──────┬──────┘
//	       │
//	┌──────▼──────┐
//	│  Execute    │  RunE in command — RuntimeProvider builds heavy
//	│             │  state lazily; use-cases consume the typed runtime.
//	└─────────────┘
//
// On every fresh command invocation in long-lived process scenarios (REPL,
// daemon), AppContext.BeginCommand re-binds the resolver and unfreezes the
// context for a new pass; the registry is rebuilt from scratch so plugin
// instances do not leak state between command runs.
//
// # Thread-safety contract
//
// AppContext is safe for concurrent reads and writes. Accessors take an
// internal RWMutex; the framework owns mutators (Update, SetResolver,
// BeginCommand, Freeze) and exposes them only to its own startup path.
// Plugins should:
//
//   - Treat ctx.Config() and any field returned by accessors as read-only;
//     mutating returned pointers may race with framework rebinds.
//   - Treat ctx.Resolver() as never-nil (returns a no-op resolver when no
//     real one is bound) and idempotent; capability lookups can run from
//     any goroutine.
//   - Treat plugin-local Config (BasePlugin[C].cfg) as mutable only via
//     FlagOverridable, ideally before RunE consumes it.
//
// Plugin factories (the function passed to registry.RegisterFactory) MUST
// be pure: the catalog calls them once at startup for the prototype and
// once per command for the per-run plugin instance.
//
// # Capability discovery
//
// Use registry.ByCapabilityFrom[T](resolver) to enumerate plugins
// implementing a capability interface inside a plugin's own logic. The
// canonical capabilities (CI provider, change detector, KV/blob caches,
// pipeline contributions, preflights) are pre-resolved by the Resolver
// interface — plugins should call those typed methods rather than
// type-asserting from raw plugin lists.
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
// tfupdate are the canonical producers. The contract test suite for blob
// backends lives at pkg/cache/blobcache/contracttest.
package plugin
