// Package plugin provides the compile-time plugin system for TerraCi.
//
// # Package layout
//
// The plugin system is organized into a small set of public packages:
//
//   - pkg/plugin           — core interfaces, BasePlugin[C], AppContext, EnablePolicy
//   - pkg/plugin/cliout    — public command output helpers (Format, ParseFormat, WriteJSON)
//   - pkg/plugin/registry  — factory catalog, command binding lookup, resolver, and lifecycle facades
//   - pkg/plugin/initwiz   — init wizard types (StateMap, StateKey, InitContributor, InitGroup)
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
//   - commands.go     — CommandProvider with CommandSpec definitions and thin flag/request parsing
//   - runtime.go      — plugin-local lazy immutable runtime builder
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
//	│  Configure  │  ConfigLoader.DecodeAndSet — config.ExtensionDocument
//	│             │  decoded into BasePlugin[C]'s private config copy.
//	└──────┬──────┘
//	       │
//	┌──────▼──────┐
//	│  Preflight  │  Preflightable.Preflight(ctx, appCtx)
//	│             │  Cheap validation only. No network, no heavy state.
//	│             │  Skipped only through typed runflow.CommandPolicy.
//	└──────┬──────┘
//	       │
//	┌──────▼──────┐
//	│  Execute    │  RunE in command — plugin-local builders create
//	│             │  heavy state lazily for typed use-cases.
//	└─────────────┘
//
// AppContext is constructed once per command run by the CLI runflow. The
// framework binds a command-scoped CommandContext to cobra through a
// CommandBinding so plugin RunE callbacks can retrieve AppContext plus planning
// state through CommandPlugin[T]. AppContext itself is runtime context only —
// it does not carry command lookup or pipeline contributions. It is immutable:
// plugins receive a snapshot of Config / WorkDir / ServiceDir / narrow
// resolver accessors that do not change for the duration of the command.
//
// # Command boundary
//
// Command setup is framework-owned: cobra flags feed the CLI runflow, which
// loads config, decodes plugin config, runs preflight, collects contributions,
// and binds a CommandBinding. Plugin command handlers should stay thin:
// resolve CommandContext plus the command-scoped plugin with CommandPlugin[T],
// read runtime state through cmdCtx.AppContext(), call RequireEnabled for
// ConfigLoader-backed plugins, parse cobra flags into a typed request, then
// hand the request to the plugin use-case. CommandPlugin and RequireEnabled
// return typed errors (CommandBindingError and DisabledPluginError) so tests
// can use errors.As. The canonical plugin command flow is:
//
//	cobra flags -> typed Request -> immutable Runtime -> use-case Result
//	    -> artifact persistence -> output renderer
//
// Plugin-local runtime builders should build immutable dependencies and
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
// # Pipeline IR boundary
//
// Framework code plans projects through workflow.PlanProject, derives the
// Terraform/OpenTofu runtime once, and converts that result into a
// provider-agnostic immutable IR with pipeline.BuildProjectIR. CI providers
// and local execution are IR consumers only: provider generator factories take
// NewGenerator(*pipeline.IR), not AppContext, and read immutable pipeline.Job
// values through getters such as IR.Jobs, Job.Operation, and
// Operation.Terraform. Terraform binary, init behavior, and execution.env are
// stored on Terraform jobs and operations through pipeline.TerraformJobConfig;
// providers should not inject implicit global binary variables or re-derive
// runtime settings from config snapshots. Local execution flow follows the
// same rule: orchestration derives terraformrun.Profile and builds IR, while
// runner packages receive runtime options and immutable jobs only. Providers
// that need barrier groups use pipeline.Schedule, whose JobGroup values expose
// read-only Name, Jobs, and JobCount accessors. Provider job builders should
// take pipeline.Job values, not job pointers. External plugin authors should
// not construct IR, Job, Operation, or TerraformOperation literals or depend
// on module job naming. Tests and advanced in-process tooling can use
// pkg/pipeline/pipelinetest for validated synthetic fixtures.
//
// # Execution result and diagnostics boundary
//
// pkg/execution.Result, JobResult, GroupResult, and JobEvent are immutable value
// objects. Local execution consumers read results through getters, Result.Stats,
// and Result.Failed; runners and event sinks receive pipeline.Job/event values,
// not mutable job pointers. Failed jobs are surfaced as execution.ExecutionError
// while the partial Result remains available, and produced artifacts are exposed
// as typed pipeline.Artifact values. Tests should use
// pkg/execution/executiontest for result fixtures instead of struct literals.
// Non-fatal warnings and degraded-mode notes should use pkg/diagnostic.List
// rather than raw []string or []error warning channels.
//
// # CI provider output boundary
//
// CI provider generators should treat pipeline.IR as the only provider input,
// then build provider-local immutable documents through their own builders.
// The canonical flow is:
//
//	workflow.PlanProject -> pipeline.BuildProjectIR -> provider document builder
//	    -> immutable provider document -> ToYAML
//
// ToYAML is the only place provider output should become raw YAML/maps. Tests
// should use semantic read helpers on provider documents, such as Job(name),
// JobNames, HasNeed, Steps, Needs, Env, and Variables, instead of indexing
// mutable job maps. Provider documents should not expose one-shot constructors
// or map-shaped Jobs accessors.
//
// # Init wizard boundary
//
// InitContributor implementations return validated groups and typed config
// through initwiz.NewInitGroup / initwiz.NewInitContribution. The canonical
// flow is:
//
//	registry -> initflow.New -> DefaultState/ApplyOverrides
//	    -> StateMap + StateKey[T] -> typed config struct
//	    -> initwiz.NewInitContribution -> config.ExtensionValue
//	    -> initflow.BuildConfig
//
// InitGroups returns ([]initwiz.InitGroup, error); return constructor errors
// instead of panicking. Returning nil, nil from BuildInitConfig is the only
// normal way to skip an optional init contribution. Do not build extension
// config with loose maps or construct InitContribution, InitGroup, or
// InitField directly; config.NewExtensionValue owns YAML node encoding, key
// validation, and defensive copies. Init fields are value objects built with
// initwiz.NewStringField, NewBoolField, or NewSelectField; plugin code should
// not read wizard state through raw string keys. The terraci command package
// owns only cobra flags, TUI rendering, preview rendering, and file writes.
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
//   - plugintest.AssertRuntimeBuilder[T] — verifies plugin-local lazy runtime
//     construction through a typed builder closure.
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
//     ctx.Config(). Snapshot accessors return defensive copies; production
//     code should not call MutableCopy except in explicit compatibility
//     adapters that need an isolated mutable configuration.
//   - Treat ctx.CIResolver(), ctx.ChangeDetectorResolver(),
//     ctx.KVCacheResolver(), and ctx.BlobStoreResolver() as never-nil. They
//     return no-op resolver behavior when no real resolver is bound and are
//     safe to call from any goroutine.
//   - Implement Clone() C on plugin config types embedded in BasePlugin[C].
//     BasePlugin.Config(), SchemaConfig(), DecodeAndSet(), and SetTypedConfig()
//     all use defensive copies; mutating Config() output never changes plugin
//     state.
//
// Plugin factories (the function passed to registry.RegisterFactory) MUST
// be pure: the catalog calls them once at startup for the prototype and
// once per command for the per-run plugin instance.
//
// # Capability discovery
//
// AppContext exposes narrow typed capability resolvers only. Plugins should
// call the resolver accessor for the capability they need instead of depending
// on broad service-locator contracts or looking up concrete plugin names.
// Framework code owns raw plugin enumeration inside pkg/plugin/registry; CLI flows consume
// registry lifecycle facades and snapshots instead of capability slices.
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
// results into typed ci.RenderBlock/ci.RenderValue values, build reports with
// ci.NewRenderedReport, and persist raw results plus the report through
// ci.PublishArtifacts. That helper always preserves raw results and removes
// stale reports when report construction fails or intentionally returns nil.
// Non-plan producers may create an ArtifactRun without PlanResults; that is
// explicit degraded mode. Consumers should load through
// ci.ReportReader/ReportStore and call ci.SelectCurrentReports before
// rendering.
// ReportSection is a value object: external plugins should not construct
// section JSON, RenderBlock, RenderTable, or RenderValue payloads manually.
// Use constructors such as ci.NewTableBlock, ci.RenderStatus, ci.RenderMoney,
// and ci.RenderModulePath so ci.NewRenderedReport can publish the current
// versioned rendered payload schema. Consumers should use ci.DecodeRenderSection
// or plugins/internal/reportrender instead of importing producer-specific
// domain structs. Markdown/CLI presentation remains centralized in
// plugins/internal/reportrender. The contract test suite for blob backends lives at
// pkg/cache/blobcache/contracttest.
// Policy plugins keep raw OPA/JSON maps at the OPA adapter boundary: use-cases
// pass typed policyinput.Envelope values and consume typed policy results.
//
// Pipeline contributions are value objects too. Producers should build jobs
// with pipeline.NewPluginCommandJob or pipeline.NewContributedJob and wrap
// them with pipeline.NewContribution. Consumers use Contribution.Jobs() and
// ContributedJob getters; direct struct literals are not part of the SDK.
package plugin
