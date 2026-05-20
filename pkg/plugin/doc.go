// Package plugin provides the compile-time plugin system for TerraCi.
//
// # Package layout
//
// The plugin system is organized into a small set of public packages:
//
//   - pkg/plugin           — core interfaces, BasePlugin[C], AppContext, EnablePolicy, RuntimeProvider
//   - pkg/plugin/cliout    — public command output helpers (Format, ParseFormat, WriteJSON)
//   - pkg/plugin/registry  — factory catalog and per-command Registry capability resolution
//   - pkg/plugin/initwiz   — init wizard types (StateMap, InitContributor, InitGroupSpec)
//
// Plugin-author contract tests live in pkg/plugin/plugintest. Helpers shared
// between built-in CI provider plugins (gitlab, github, future
// Bitbucket/Jenkins/Azure DevOps) live in plugins/internal/ciplugin. Shared
// report rendering for plugin-owned ci.Report payloads lives in
// plugins/internal/reportrender. The plugins/internal packages are not part of
// the public API.
//
// # Plugin file convention
//
// Each command-oriented plugin with a typed runtime boundary (cost, policy,
// summary, tfupdate) keeps one file per capability so the file list reads as a
// capability index:
//
//   - plugin.go       — registration shell + typed BasePlugin[C] config
//   - lifecycle.go    — cheap Preflight checks only (no network, no FS scan)
//   - commands.go     — CommandProvider with thin cobra/request parsing
//   - runtime.go      — lazy immutable RuntimeProvider implementation
//   - usecases.go     — typed Request/Result orchestration over runtime
//   - pipeline.go     — PipelineContributor (pipeline DAG jobs; return builder errors)
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
//	│             │  YAML node decoded into BasePlugin[C]'s private config copy.
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
// attached to cmd.Context() so plugin RunE callbacks can retrieve it through
// CommandPlugin[T]. It is immutable — plugins receive a snapshot of Config /
// WorkDir / ServiceDir / Resolver / pipeline contributions that does not
// change for the duration of the command.
//
// # Command boundary
//
// Command handlers should stay thin: resolve the command-scoped plugin with
// CommandPlugin[T], call RequireEnabled for ConfigLoader-backed plugins, parse
// cobra flags into a typed request, then hand the request to the plugin
// use-case. CommandPlugin and RequireEnabled return typed errors
// (CommandBindingError and DisabledPluginError) so tests can use errors.As.
// The canonical flow is:
//
//	cobra flags -> typed Request -> immutable Runtime -> use-case Result
//	    -> artifact persistence -> output renderer
//
// RuntimeProvider implementations should build immutable dependencies and
// normalized config only. Command-specific overrides belong in the request, so
// repeated command invocations cannot leak mutable runtime state.
//
// # Pipeline contribution boundary
//
// PipelineContributor implementations return (*pipeline.Contribution, error).
// Build jobs with pipeline.NewPluginCommandJob or pipeline.NewContributedJob,
// wrap them with pipeline.NewContribution, and return any builder error. A
// nil contribution with nil error is invalid; optional jobs belong behind
// PipelineContributionGate so the framework can distinguish "not enabled for
// this run" from "broken contribution".
//
// # Init wizard boundary
//
// InitContributor implementations return typed config through
// initwiz.NewInitContribution. The canonical flow is:
//
//	StateMap -> typed config struct -> initwiz.NewInitContribution
//	    -> config.ExtensionValue -> config.Build
//
// Returning nil, nil is the only normal way to skip an optional init
// contribution. Do not build extension config with loose maps or construct
// InitContribution directly; config.NewExtensionValue owns YAML node encoding,
// key validation, and defensive copies.
//
// # SDK contract tests
//
// External plugin authors should copy the contract-style tests from
// pkg/plugin/plugintest and pkg/ci/citest instead of writing ad-hoc tests for
// SDK behavior. The canonical helpers are:
//
//   - plugintest.AssertBaseConfigPlugin[C] — verifies Clone() C and
//     BasePlugin defensive copies.
//   - plugintest.AssertCommandBinding[T] — verifies CommandPlugin[T] command
//     lookup and stable CommandBindingError reasons.
//   - plugintest.AssertRequireEnabled — verifies DisabledPluginError behavior.
//   - plugintest.AssertRuntimeProvider[T] — verifies lazy RuntimeProvider
//     construction and RuntimeAs[T].
//   - plugintest.AssertPipelineContributor — verifies deterministic generic
//     contribution shape.
//   - plugintest.AssertPreflightable, AssertInitContributor,
//     AssertVersionProvider, AssertKVCacheProvider, AssertBlobStoreProvider,
//     AssertChangeDetector, and AssertCIProvider — verify the remaining
//     capability contracts through focused, closure-based test fixtures.
//   - citest.AssertRenderedReportContract — verifies ci.NewRenderedReport
//     output validates, decodes through ci.DecodeRenderSection, and renders.
//   - citest.AssertPublishArtifactsContract — verifies ci.PublishArtifacts
//     replacement semantics with a recording ArtifactWriter.
//
// # Thread-safety contract
//
// AppContext fields are written exactly once at construction, so concurrent
// reads from any goroutine are safe without synchronization. Plugins
// should:
//
//   - Read project config through the immutable config.Snapshot returned by
//     ctx.Config(). Snapshot accessors return defensive copies; use
//     MutableCopy only for legacy pointer-shaped APIs.
//   - Treat ctx.Resolver() as never-nil (returns NoopResolver{} when no
//     real one is bound) and idempotent; capability lookups can run from
//     any goroutine.
//   - Implement Clone() C on plugin config types embedded in BasePlugin[C].
//     BasePlugin.Config(), NewConfig(), DecodeAndSet(), and SetTypedConfig()
//     all use defensive copies; mutating Config() output never changes plugin
//     state.
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
//     with plugin-agnostic domain contracts, such as workflow.ChangeDetector,
//     owned by their core package.
//   - shared types in pkg/ci (Report, ReportSection, PlanResult, …)
//   - the ci.ReportStore on AppContext, which owns both file-backed artifacts
//     ({producer}-report.json) and in-process report exchange
//
// summary is the canonical consumer of report artifacts; cost/policy/
// tfupdate are the canonical producers. Plan-aware producers should carry the
// scanned ci.PlanResultCollection into ci.NewArtifactRun, convert domain
// results into ci.RenderBlock values, build reports with ci.NewRenderedReport,
// and persist raw results plus the report through ci.PublishArtifacts. That
// helper always preserves raw results and removes stale reports when report
// construction fails or intentionally returns nil. Non-plan producers may
// create an ArtifactRun without PlanResults; that is explicit degraded mode.
// Consumers should load through ci.ReportReader/ReportStore and call
// ci.SelectCurrentReports before rendering.
// ReportSection is a value object: external plugins should not construct
// section JSON or payloads manually, and direct field access is intentionally
// unavailable. Consumers should use ci.DecodeRenderSection or
// plugins/internal/reportrender instead of importing producer-specific domain
// structs. The contract test suite for blob backends lives at
// pkg/cache/blobcache/contracttest.
//
// Pipeline contributions are value objects too. Producers should build jobs
// with pipeline.NewPluginCommandJob or pipeline.NewContributedJob and wrap
// them with pipeline.NewContribution. Consumers use Contribution.Jobs() and
// ContributedJob getters; direct struct literals are not part of the SDK.
package plugin
