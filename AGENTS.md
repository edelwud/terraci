# TerraCi

CLI tool for analyzing Terraform projects, building dependency graphs, generating CI pipelines, and estimating cloud costs. Extended via compile-time plugin system.

## Build & Test

```bash
task build          # Build terraci + xterraci → build/ (incremental)
task test           # Run tests with race detection and coverage
task test:short     # Short tests
task lint           # golangci-lint + go vet
task fmt            # Format code
task build:install  # Install both to $GOPATH/bin
task build:all      # Cross-compile for all platforms
task clean          # Clean build artifacts
task dev:setup      # Install all dev tools (idempotent)
task dev:run        # Build and run terraci (-- args passthrough)
task dev:watch      # Watch files and rebuild on change
task dev:security   # Run govulncheck
task ci:pr          # Full PR validation (lint + test + integration)
```

## Project Structure

```
cmd/terraci/
├── main.go                     # Entry point — blank-imports all built-in plugins
└── cmd/
    ├── app.go                  # Thin CLI state holder for flags/version/current registry
    ├── root.go                 # NewRootCmd(), runflow.Prepare binding, dynamic commands
    ├── generate.go             # Pipeline generation command presentation over generateflow
    ├── graph.go                # Dependency graph command presentation over graphflow
    ├── validate.go             # Config/project validation presentation over validateflow
    ├── filters.go              # filterFlags struct — shared filter flags, mergedFilterOpts()
    ├── init.go                 # Config initialization CLI/file I/O (--ci mode delegates to initflow)
    ├── init_tui.go             # Interactive TUI presentation over initflow display groups
    ├── schema.go               # JSON schema (includes extension schemas)
    ├── version.go              # Version info via VersionProvider plugins
    ├── completion.go           # Shell completion
    └── man.go                  # Man page generation

cmd/xterraci/
├── main.go                     # Entry point
└── cmd/
    ├── root.go                 # NewRootCmd(version, commit, date)
    ├── build.go                # xterraci build — custom binary builder
    ├── list.go                 # xterraci list-plugins
    ├── version.go              # xterraci version
    ├── completion.go           # Shell completion
    ├── man.go                  # Man pages
    ├── builder.go              # Build orchestration: temp dir, codegen, go build
    ├── codegen.go              # Generates main.go with plugin imports
    ├── plugins.go              # Built-in plugin import paths + validation
    └── *_test.go

cmd/terraci/internal/
├── generateflow/               # Generate command orchestration: projectflow → BuildProjectIR → provider generator
├── graphflow/                  # Graph command orchestration and graph render formats
├── initflow/                   # Typed init wizard orchestration: defaults, plugin groups, config build
├── projectflow/                # Thin runflow adapter over workflow.PlanProject
├── runflow/                    # Typed command lifecycle: config load, plugin decode, preflight, contributions
├── schemaflow/                 # Schema command orchestration
├── validateflow/               # Validate command graph diagnostics and pass/fail decision
└── versionflow/                # Version command orchestration

pkg/                            # Public API — importable by external plugins (plugin-agnostic core + plugin SDK)
├── plugin/                     # Core plugin SDK — interfaces, BasePlugin, AppContext
│   ├── plugin.go               # Plugin
│   ├── lifecycle.go            # ConfigLoader, Preflightable
│   ├── commands.go             # CommandProvider, VersionProvider
│   ├── ci_provider.go          # EnvDetector, CIInfoProvider, PipelineGeneratorFactory, CommentServiceFactory, ResolvedCIProvider
│   ├── cache.go                # KVCacheProvider, BlobStoreProvider (single NewBlobStore with options)
│   ├── change.go               # ChangeDetectionProvider
│   ├── contribution.go         # PipelineContributor
│   ├── base.go                 # BasePlugin[C] generic embedding
│   ├── enable.go               # EnablePolicy enum
│   ├── context.go              # AppContext + AppContextOptions constructor
│   ├── command_binding.go      # CommandPlugin[T], AppContextFromCommand, RequireEnabled + typed command errors
│   ├── cliout/                 # Public plugin command output helpers (Format, ParseFormat, WriteJSON)
│   ├── runtime.go              # RuntimeProvider + RuntimeAs() + BuildRuntime[T]()
│   ├── resolver.go             # Narrow resolver interfaces + aggregate implementation contract
│   ├── noop_resolver.go        # default no-op resolver behavior
│   ├── registry/               # Plugin factory catalog + per-command Registry
│   │   ├── registry.go         # RegisterFactory(), New(), Catalog, Registry + typed framework views
│   │   └── resolve.go          # Registry.ResolveCIProvider/ResolveChangeDetector/Resolve*Provider/CollectContributions/PreflightsForStartup
│   ├── initwiz/                # Init wizard state + types
│   │   ├── state.go            # StateMap + typed StateKey[T] form state for huh bindings
│   │   └── types.go            # InitContributor, InitGroupSpec, constructor-built InitField value objects
│   └── plugintest/             # Plugin-author SDK contract tests + mock doubles + NoopResolver
├── pipeline/                   # Plugin-agnostic pipeline IR
│   ├── types.go                # IR, Job, Operation, TerraformOperation, Contribution, ContributedJob
│   ├── contribution.go         # Contribution/ContributedJob value-object builders + getters
│   ├── builder.go              # BuildProjectIR(...) — constructs provider-agnostic pipeline IR
│   ├── pipeline.go             # Generator interface (Generate()/DryRun() — IR bound at construction)
│   ├── common.go               # Internal job planning helpers + IR.DryRun
│   ├── jobs.go                 # JobKind + IR module counts
│   ├── env.go                  # ModuleEnvVars
│   ├── scripts.go              # TerraformJobConfig, NewPlanOperation, NewApplyOperation (IR construction only)
│   └── cishell/                # Shell renderer — keeps shell-specific knowledge out of the IR
│       └── render.go           # RenderOperation: pipeline.Operation → POSIX shell command lines
├── terraformrun/               # Terraform/OpenTofu runtime profile from immutable config snapshots
├── ci/                         # Plugin-agnostic CI types
│   ├── report.go, report_store.go, report_types.go, report_freshness.go, report_validation.go, section.go
│   │                           #   Report envelope, ReportStore, ArtifactContext/ArtifactRun, ReportSection, versioned typed RenderSection/RenderBlock/RenderValue payloads, NewRenderedReport/NewRenderedSection, SelectCurrentReports + validation
│   ├── plan.go                 # PlanResult (canonical for both in-memory + persisted), PlanResultCollection, PlanStatus
│   ├── service.go              # CommentService
│   └── shared.go               # Image, CommentMarker
├── cache/blobcache/            # Blob store contract + cache layer (owns Store/Meta/Object/PutOptions/Info/Inspector/Describer/HealthChecker, Describe, Check, Cache, Policy)
├── config/
│   ├── types_config.go         # Config (service_dir, structure, exclude, include, library_modules, extensions map[string]yaml.Node)
│   ├── clone.go, snapshot.go   # Deep-copy API and immutable Snapshot read model
│   ├── builder.go              # Build(BuildOptions) + typed ExtensionValue/ExtensionSet
│   ├── extension.go            # (*Config).Extension(key, target) — opaque section decoder
│   ├── pattern.go              # ParsePattern, PatternSegments
│   ├── schema.go               # GenerateJSONSchema(extensionSchemas)
│   ├── io.go                   # Load, LoadOrDefault, Save
│   ├── defaults.go             # DefaultConfig()
│   └── validation.go           # Validate
├── discovery/                  # Module, Scanner, ModuleIndex (slim: All/ByID/ByPath)
├── parser/                     # Public parser facade + shared model
│   ├── parser.go               # ParseModule() facade over internal moduleparse pipeline
│   ├── dependency.go           # DependencyExtractor facade
│   ├── model_aliases.go        # Re-exports for ParsedModule/RequiredProvider/LockedProvider/ModuleCall/RemoteStateRef/LibraryDependency/ModuleDependencies
│   ├── model/                  # Stable shared parser model used by facade + internals
│   └── internal/               # Layered parser internals (moduleparse, dependency, source, extract, resolve, evalctx, exprfast, deps, testutil)
├── planresults/                # PlanResultCollection scanner — reads Terraform plan.json
├── diagnostic/                 # Plugin-agnostic non-fatal diagnostics value model
├── execution/                  # Plugin-agnostic local execution
│   ├── executor.go             # Executor.Execute(ctx, *pipeline.IR) over immutable pipeline.Job values
│   ├── scheduler.go            # DefaultScheduler.Schedule(*pipeline.IR) — barrier-grouped JobGroups
│   ├── event.go                # JobEvent value object for progress/event sinks
│   ├── results.go              # Immutable Result/JobResult/GroupResult + stats and produced artifacts
│   ├── errors.go               # ExecutionError wrapping failed job causes
│   ├── worker_pool.go, workspace.go, config.go
├── graph/                      # DependencyGraph, algorithms, visualization
├── filter/                     # Public surface: Options + Apply + Flags + ParseSegmentFilters (concrete filter types unexported)
├── workflow/                   # Module discovery, filtering, graph building, target resolution, plugin-agnostic ChangeDetector request/result contract
├── errors/                     # Typed errors (ConfigError, ScanError, ParseError, NoModulesError)
└── log/                        # Thin wrapper over caarlos0/log

plugins/                        # Built-in plugins — one file per capability
├── gitlab/
│   ├── plugin.go               # init, BasePlugin[*Config] embed
│   ├── lifecycle.go            # Preflightable (cheap MR context detection)
│   ├── generator.go            # EnvDetector + CIInfoProvider + PipelineGeneratorFactory(ctx, *pipeline.IR) + CommentServiceFactory
│   ├── init_wizard.go          # InitContributor
│   └── internal/               # config, generator, MR service, domain types
│       └── generate/
│           ├── buildir.go      # BuildPipelineIR helper for tests (IR construction with plugin settings)
│           └── generator.go    # Generator stores *pipeline.IR; Generate()/DryRun() with no args
├── github/
│   ├── plugin.go               # init, BasePlugin[*Config] embed
│   ├── lifecycle.go            # Preflightable (cheap PR context detection)
│   ├── generator.go            # EnvDetector + CIInfoProvider + PipelineGeneratorFactory(ctx, *pipeline.IR) + CommentServiceFactory
│   ├── init_wizard.go          # InitContributor
│   └── internal/generate/      # IR-bound generator + buildir.go test helper
├── cost/
│   ├── plugin.go               # init, BasePlugin[*CostConfig] embed
│   ├── lifecycle.go            # Preflightable (cheap config/cache validation)
│   ├── commands.go             # CommandProvider (terraci cost)
│   ├── runtime.go              # RuntimeProvider (lazy estimator construction)
│   ├── usecases.go             # Request/result usecase orchestration
│   ├── pipeline.go             # PipelineContributor
│   ├── init_wizard.go          # InitContributor
│   ├── output.go               # CLI rendering helpers
│   ├── report.go               # CI report assembly via ci.NewRenderedReport
│   └── internal/               # (package costengine) — layered cost estimation engine
│       ├── engine/, runtime/, model/, results/, cloud/{aws,awskit}, resourcedef/, resourcespec/, costutil/, pricing/, contracttest/, enginetest/
├── policy/
│   ├── plugin.go, lifecycle.go, commands.go, runtime.go, usecases.go, pipeline.go, version.go, init_wizard.go, output.go, report.go
│   └── internal/               # (package policyengine) OPA engine, checker, sources
├── tfupdate/
│   ├── plugin.go, lifecycle.go, commands.go, runtime.go, usecases.go, pipeline.go, output.go, report.go, init_wizard.go
│   └── internal/               # (package tfupdateengine) planner, lockfile, sourceaddr, registrymeta, usecase, registryclient, tffile, tfwrite
├── summary/
│   ├── plugin.go, commands.go, usecases.go, pipeline.go, init_wizard.go, output.go
│   └── internal/summaryengine/ # config, renderer, report_loader, labels, usecase orchestration
├── localexec/
│   ├── plugin.go               # init, Plugin struct
│   ├── commands.go             # CommandProvider (terraci local-exec with plan/run only)
│   ├── contract.go             # Public stable NewExecutor(...) boundary for in-process callers
│   └── internal/
│       ├── executor.go         # Thin adapter from public contract to internal flow
│       ├── flow/               # Use-case orchestration: PlanProject → BuildProjectIR → execute → render
│       ├── render/             # Progress output and local CLI rendering
│       ├── runner/             # Shell/Terraform runners + DAG job orchestration
│       ├── spec/               # Internal validated execute request/mode types
│       └── reports/            # Current report loading/selection for local summaries
├── diskblob/
│   ├── plugin.go               # init, BasePlugin[*Config] embed, BlobStoreProvider (single NewBlobStore with options)
│   ├── config.go               # Backend config (enabled, root_dir)
│   ├── store.go                # Blob store construction helpers
│   ├── home.go                 # Home/service-dir root resolution
│   └── internal/               # Filesystem-backed blob store implementing blobcache.Store
├── inmemcache/
│   ├── plugin.go               # init, BasePlugin[*Config] embed, KVCacheProvider
│   ├── cache.go                # Process-local in-memory cache implementation
│   └── *_test.go
├── git/
    ├── plugin.go               # init, Plugin struct (no config, no BasePlugin)
    ├── lifecycle.go            # Preflightable (cheap repo detection)
    ├── detect.go               # ChangeDetectionProvider
    └── internal/gitclient/     # client, detector, diff
└── internal/
    ├── ciplugin/               # Shared CI-provider helpers
    └── reportrender/           # Shared markdown/CLI renderer for ci.Report render-ready payloads

internal/                       # Private — only terraform eval
└── terraform/
    ├── eval/                   # NewContext(), 30+ Terraform functions
    └── plan/                   # ParseJSON, ResourceChange, AttrDiff
```

## Plugin System

### Architecture

Compile-time plugins via `init()` + blank import (Caddy/database-sql pattern). Plugins register factories via `registry.RegisterFactory()`, and `cmd/terraci/internal/runflow` creates a fresh `*registry.Registry` for each command run. `*Registry` implements the narrow plugin resolver interfaces, while framework capability discovery stays inside runflow/schemaflow/versionflow or registry tests. Core types (interfaces, BasePlugin, AppContext) live in `pkg/plugin`; plugin catalog and per-command registries live in `pkg/plugin/registry`; init wizard types in `pkg/plugin/initwiz`.

The core `pkg/` tree is **plugin-agnostic** — no package outside `pkg/plugin` imports the plugin SDK. Plugin extensibility hangs entirely off `pkg/plugin`'s capability interfaces.

### Plugin File Convention

Each feature/plugin follows one-file-per-capability where it applies, with runtime-heavy plugins also using a lazy runtime layer. Backend plugins such as `diskblob` and `inmemcache` are intentionally smaller and only implement their relevant provider interfaces:
- `plugin.go` — init(), Plugin struct with BasePlugin[C] embedding
- `lifecycle.go` — Preflightable
- `commands.go` — CommandProvider with cobra definitions; parse flags into typed requests and resolve via `plugin.CommandPlugin`
- `runtime.go` — RuntimeProvider for lazy immutable dependency construction
- `usecases.go` — typed Request/Result orchestration over runtime
- `generator.go` — EnvDetector + CIInfoProvider + PipelineGeneratorFactory + CommentServiceFactory
- `pipeline.go` — PipelineContributor(ctx) (no self-check, framework filters)
- `init_wizard.go` — initwiz.InitContributor (uses package-local initwiz.StateKey values and field constructors)
- `version.go` — VersionProvider
- `output.go` — Rendering/formatting helpers
- `report.go` — CI report assembly
- `detect.go` — ChangeDetectionProvider

### Plugin Lifecycle

```
1. Register    — init() calls registry.RegisterFactory() with a Plugin factory
2. Configure   — ConfigLoader.DecodeAndSet() for plugins with a config section under extensions:
3. Preflight   — Preflightable.Preflight() performs cheap validation/env detection
4. Bind        — runflow builds immutable AppContext/Prepared and attaches it to command context
5. Execute     — Commands parse flags into typed requests; use-cases lazily build RuntimeProvider runtimes as needed
```

### Capability Interfaces

| Interface | Purpose | Implemented by |
|-----------|---------|----------------|
| `Plugin` | Base: Name(), Description() | all |
| `ConfigLoader` | Config section under `extensions:` + IsEnabled() via EnablePolicy | gitlab, github, cost, policy, summary, tfupdate |
| `CommandProvider` | CLI subcommands | cost, policy, summary, tfupdate, localexec |
| `Preflightable` | Cheap startup validation / env detection | gitlab, github, cost, policy, git, tfupdate |
| `RuntimeProvider` | Lazy command-time runtime construction | cost, policy, tfupdate |
| `EnvDetector` | CI environment detection | gitlab, github |
| `CIInfoProvider` | Provider name, pipeline ID, commit SHA | gitlab, github |
| `PipelineGeneratorFactory` | Pipeline generator creation — `NewGenerator(ctx, *pipeline.IR)` | gitlab, github |
| `CommentServiceFactory` | MR/PR comment service creation | gitlab, github |
| `VersionProvider` | Version info contributions | policy |
| `ChangeDetectionProvider` | VCS change detection | git |
| `KVCacheProvider` | Named key/value cache backend resolution | inmemcache |
| `BlobStoreProvider` | Named blob/object store backend (`NewBlobStore(ctx, appCtx, opts)`) | diskblob |
| `InitContributor` | Init wizard form fields + config building | gitlab, github, cost, policy, summary, tfupdate |
| `PipelineContributor` | Pipeline steps/jobs via Contribution | cost, policy, summary, tfupdate |

### BasePlugin[C] Generic Embedding

Plugins with config embed `BasePlugin[C]`; `C` must implement `Clone() C`.
`BasePlugin` stores and returns defensive copies, so mutating `Config()` output
never changes plugin state. It auto-implements:
- `Name()`, `Description()`, `ConfigKey()`, `NewConfig()`, `DecodeAndSet()`, `IsConfigured()`, `IsEnabled()`, `Config()`, `Reset()`
- `EnablePolicy` controls enabled semantics: `EnabledWhenConfigured` (gitlab/github), `EnabledExplicitly` (cost/policy/tfupdate), `EnabledByDefault` (summary/diskblob/inmemcache). Bare plugins such as `git` are active by registration and do not implement `ConfigLoader`.

### AppContext

Construction goes through an options struct — `plugin.NewAppContext(plugin.AppContextOptions{Config, WorkDir, ServiceDir, Version, Reports, Resolver})` — every field is optional. `Reports` is a `ci.ReportStore`; it defaults to a file-backed store when `ServiceDir` is set, otherwise an in-memory store. Resolver access is narrow and **never nil** through `ctx.CIResolver()`, `ctx.ChangeDetectorResolver()`, `ctx.KVCacheResolver()`, and `ctx.BlobStoreResolver()`; when no resolver is bound, no-op resolvers return sentinel errors.

`AppContext.Config()` returns an immutable `config.Snapshot`. Access config through
snapshot accessors (`ServiceDir()`, `Structure()`, `Execution()`, etc.).
Production plugin code should not call `MutableCopy()`; keep it for tests or
explicit compatibility adapters that need an isolated mutable config.

### Command Boundary

Command handlers should stay as a thin boundary: resolve `appCtx` and the command-scoped plugin with `plugin.CommandPlugin[T]`, gate configured plugins with `plugin.RequireEnabled`, parse cobra flags into a typed request, then call a usecase method. The canonical shape is:

```
cobra flags → typed Request → immutable Runtime → usecase Result → artifact persistence → output
```

RuntimeProvider implementations should hold normalized config and constructed dependencies only. Command-specific values such as `--module`, `--output`, `--write`, timeouts, and override flags belong in request structs, not mutable runtime fields.

### SDK Contract Tests

Plugin-author tests should reuse the public contract kit instead of duplicating SDK behavior:
- `pkg/plugin/plugintest`: `AssertBaseConfigPlugin`, `AssertCommandBinding`, `AssertRequireEnabled`, `AssertRuntimeProvider`, `AssertPipelineContributor`, plus capability contracts for preflight, init wizard, version info, KV/blob providers, change detection, and CI providers.
- `pkg/ci/citest`: `AssertRenderedReportContract`, `AssertPublishArtifactsContract`, `RecordingArtifactWriter`, and rendered-section/report builders.

Built-in plugins and examples keep domain-specific tests local, but SDK boundary behavior is asserted through these helpers so third-party authors can copy the same patterns.

### Resolver

The plugin SDK exposes narrow resolver interfaces: `CIResolver`, `ChangeDetectorResolver`, `KVCacheResolver`, and `BlobStoreResolver`. `plugin.Resolver` is only the aggregate implementation contract used by framework wiring; plugin code should consume the specific AppContext resolver accessor it needs. `*registry.Registry` is the production resolver and also owns framework-only catalog operations such as contribution collection and startup preflights, which are invoked by `runflow`.

### Shared Types

`pkg/ci/` contains shared CI-domain types including provider-shared config such as `Image` (with YAML shorthand). `ci.Report` is the typed report contract shared by cost/policy/tfupdate/summary; `ci.ReportStore` owns in-memory and file-backed report persistence. Plan-aware producers carry one `PlanResultCollection` into `ci.ArtifactRun`; reports carry provenance derived from `ArtifactRun.Artifact` so local consumers can validate current artifacts. Both gitlab and github internal packages use type aliases to these.

`ci.ReportSection` is a value object for render-ready report sections. Producer plugins convert domain results into typed `ci.RenderBlock` / `ci.RenderValue` values and call `ci.NewRenderedReport(...)`; external plugin authors should not construct section JSON, `RenderBlock`, `RenderTable`, or `RenderValue` payloads manually. Use constructors such as `ci.NewTableBlock`, `ci.NewListBlock`, `ci.RenderStatus`, `ci.RenderMoney`, `ci.RenderMoneyDelta`, `ci.RenderModulePath`, and `ci.RenderCode`. The rendered payload is versioned with `schema_version` and stale unversioned artifacts must be regenerated by rerunning producer commands. Consumers use `ci.DecodeRenderSection` or `plugins/internal/reportrender` and do not import producer/plugin domain packages. Markdown/CLI rendering of these generic sections lives in `plugins/internal/reportrender`, not in producer plugins, so visible labels, money formatting, deltas, escaping, and status text stay centralized.

`ci.PlanResult` is the canonical representation of one module's plan outcome — used both in-memory and on disk; `ci.PlanResultCollection` aggregates them with a stable fingerprint.

`pkg/diagnostic` is the shared non-fatal diagnostic model. Use it instead of
raw `[]string` / `[]error` warning channels when a result needs warnings,
skips, degraded-mode notes, or report freshness messages. Diagnostics carry a
severity, stable message, optional source/module/hint, and optional wrapped
cause.

### Pipeline IR

`workflow.PlanProject(...)` produces the canonical project/target snapshot and
`pipeline.BuildProjectIR(...)` turns it into an immutable provider-agnostic IR.
Generators transform that IR through provider-local output builders and then
serialize the immutable provider document to YAML:

```
workflow.PlanProject(...) → pipeline.BuildProjectIR(...) → *pipeline.IR
  ↓
GitLab: IR → PipelineBuilder → Pipeline.ToYAML()
GitHub: IR → WorkflowBuilder → Workflow.ToYAML()
```

The IR is the **single source** for downstream consumers and a value object:
production code does not construct `IR`, `Job`, `Operation`, or
`TerraformOperation` literals. `pipeline.Generator` is constructed with an
`*IR` and `Generate() (GeneratedPipeline, error)` / `DryRun()
(*DryRunResult, error)` take no further arguments. Job-level access methods
live on `*IR`: `Jobs()`, `FindJob(name)`, `JobsByKind(kind)`,
`JobNamesByKind(kind)`, `JobForModule(kind, module)`, and
`HasDependency(job, dep)`. `pipeline.Schedule(ir)` returns immutable
barrier-group value objects with `Name()`, `Jobs()`, and `JobCount()`.
Provider output is also a value object: provider generators build documents
through provider-local builders, tests read them through semantic helpers such
as `Job(name)`, `JobNames()`, `HasNeed(job, dep)`, `Steps()`, `Needs()`, and
`Env()`, and `ToYAML()` is the only raw YAML/map boundary. Do not add
one-shot provider document constructors or job-map read APIs.

Shell rendering (`cd module && terraform init && terraform plan -out=…`) lives in `pkg/pipeline/cishell` (`cishell.RenderOperation(op)`) — never in the IR package itself. The binary comes from `terraformrun.Profile` and is stored on each `pipeline.TerraformOperation`, so providers and local execution consume the IR instead of injecting global binary environment variables.

Plugins contribute via `PipelineContributor.PipelineContribution(ctx) (*pipeline.Contribution, error)`:
- `pipeline.NewPluginCommandJob(...)` / `pipeline.NewContributedJob(...)` build validated standalone DAG jobs with typed resource inputs/outputs.
- `pipeline.NewContribution(jobs...)` builds the immutable contribution value; consumers use `Contribution.Jobs()` and job getters.
- returning `nil, nil` is invalid; use `PipelineContributionGate` for optional contribution and return real builder errors for diagnostics.

### Provider Resolution

`Registry.ResolveCIProvider()` returns `*plugin.ResolvedCIProvider` (struct wrapping EnvDetector + CIInfoProvider + PipelineGeneratorFactory + CommentServiceFactory): `TERRACI_PROVIDER` env → CI env detection → single active provider → error. Core has zero knowledge of specific providers. Commands that don't need config/preflight are marked with typed `runflow.CommandPolicy`; raw cobra annotations are a private runflow storage detail.

### Service Directory

`AppContext.ServiceDir` — resolved absolute path to project service directory for runtime file I/O. Configurable via `service_dir` in config (default `.terraci`). For pipeline artifact paths (CI templates), use `AppContext.Config().ServiceDir()` which preserves the original relative value.

## Configuration (.terraci.yaml)

```yaml
service_dir: .terraci  # optional, default

structure:
  pattern: "{service}/{environment}/{region}/{module}"

exclude: ["*/test/*"]
include: []

library_modules:
  paths: ["_modules"]

execution:
  binary: terraform
  init_enabled: true
  parallelism: 4
  env:
    TF_IN_AUTOMATION: "true"

extensions:
  gitlab:
    cache:
      enabled: true
      policy: pull-push
      paths: [ "{module_path}/.terraform/" ]

  summary:
    on_changes_only: false
    include_details: true
    labels:
      - terraform
      - "{environment}"
      - "{module}"
      - "resource:{resource_type}"

  # cost:
  #   providers:
  #     aws: { enabled: true }
  #   blob_cache:
  #     backend: diskblob
  #     ttl: "24h"

  # policy:
  #   enabled: true
  #   sources: [{ path: terraform }]
  #   on_failure: block

  # tfupdate:
  #   enabled: true
  #   target: all
  #   policy:
  #     bump: minor
  #   lock:
  #     platforms: [linux_amd64, darwin_arm64]
```

Core config: `service_dir`, `structure`, `exclude`, `include`, `library_modules`, `extensions` (opaque map). All provider/feature config under `extensions:`.

## Data Flow

### Generate pipeline
1. `runflow.Prepare(...)` loads config, decodes plugins, preflights, and gathers PipelineContributor jobs
2. `generateflow.Run(...)` delegates project discovery/targeting to `projectflow`
3. `projectflow.Run(...)` adapts `runflow.Prepared` into `workflow.PlanProject(...)`; changed-only detection runs inside workflow targeting
4. `generateflow` calls `pipeline.BuildProjectIR(...)` and binds the IR to `provider.NewGenerator(appCtx, ir)`
5. `cmd/terraci/cmd` renders dry-run output or writes generated YAML

### Summary
1. `planresults.Scan()` → PlanResultCollection
2. Load reports through `ci.ReportStore` (`{serviceDir}/*-report.json` plus any in-process published reports) and select current/degraded reports with `ci.SelectCurrentReports`
3. `summaryengine.ResolveLabels()` expands static/module/resource labels from changed or failed plan results
4. `summaryengine.ComposeCommentWithOptions()` renders markdown and embeds managed-label metadata
5. `appCtx.CIResolver().ResolveCIProvider()` → `NewCommentService()` → `UpsertComment(ctx, body)`
6. If the comment service implements `ci.ManagedLabelService`, sync labels by removing only prior TerraCI-managed labels absent from the current run and adding current labels

### Local Execution
1. `workflow.PlanProject(...)` builds the canonical filtered module/graph result and selected targets
2. `localexec/internal/flow` builds the execution IR through `pipeline.BuildProjectIR(...)`
3. `pkg/execution.Executor.Execute(ctx, ir)` schedules immutable job values with dependency-aware DAG grouping
4. `localexec/internal/runner` executes shell/tfexec jobs locally and CLI injects an execution event sink for progress logs
5. `localexec/internal/reports` loads current plugin reports through `ci.ReportStore`, applies `ci.SelectCurrentReports`, and aggregates them into a render-ready summary report
6. `localexec/internal/flow` returns a typed result with immutable `execution.Result`, optional summary report, and diagnostics; `localexec/internal/render` prints the DAG/job summary and report sections

### Init wizard
1. `initflow.New(registry)` snapshots init contributors, provider options, and deterministic display groups
2. `Flow.DefaultState()` plus `Flow.ApplyOverrides(...)` populate provider, binary, pattern, plan jobs, and summary defaults
3. `cmd/terraci/cmd` renders Basics plus `Flow.DisplayGroups()` through huh; it does not discover contributors or assemble YAML
4. `BuildInitConfig` reads typed `initwiz.StateKey` values and returns typed `initwiz.InitContribution` values from plugins
5. `Flow.BuildConfig(state)` converts contributions into a config extension set and assembles the final config

## Key Patterns

- **Plugin-agnostic core**: nothing under `pkg/` (except `pkg/plugin/...`) imports the plugin SDK. No mention of "plugin" in core package types or YAML keys.
- **Plugin-first feature surfaces**: every CI provider, cost backend, policy engine, etc. lives in `plugins/`.
- **One file per capability**: plugin.go < 30 lines; each interface in its own file
- **Compile-time extensibility**: `xterraci build --with/--without` for custom binaries
- **Pipeline IR**: `workflow.PlanProject(...)` → `pipeline.BuildProjectIR(...)` → immutable `*pipeline.IR`. The IR is the single execution input — generators and the local executor both consume `pipeline.Job` values through getters, not direct field mutation, job pointers, or manual literals.
- **Terraform runtime intent**: `config.Snapshot` → `terraformrun.Profile` → `pipeline.TerraformJobConfig` → Terraform jobs in the IR. `execution.env` is Terraform-job env only; command jobs and provider workflow globals do not receive it implicitly. `pipeline.BuildIntent` must come from `ApplyBuildIntent` or `PlanBuildIntent`, and plan jobs/artifacts are derived from apply intent plus requested resources.
- **IR-bound generators**: `PipelineGeneratorFactory.NewGenerator(ctx, *pipeline.IR)` — providers don't reach for depGraph/modules/contributions; the IR already encodes them. Provider job builders take immutable `pipeline.Job` values from `IR.Jobs()` and produce provider document jobs through provider-local builders.
- **Shell rendering separated from IR**: `pkg/pipeline/cishell.RenderOperation(op)` for shell-driven CI; the IR carries `pipeline.TerraformOperation` data only.
- **Canonical dry-run source**: dry-run stage/job counts derive from `*IR.DryRun(totalModules)`.
- **Execution result boundary**: `pkg/execution.Result`, `JobResult`, `GroupResult`, and `JobEvent` are immutable value objects. Production code reads them through getters and `Stats()`, never through struct literals or mutable fields. `JobRunner`, `WorkerPool`, and `EventSink` consume `pipeline.Job`/event values, not job pointers. Failed jobs surface as `execution.ExecutionError` while still returning the partial result, and produced artifacts are exposed as typed `pipeline.Artifact` values.
- **Preflight, then lazy runtime**: framework performs cheap startup validation; heavy plugin state is built lazily inside RuntimeProvider/use-cases. Runtime must be command-agnostic; CLI overrides live in typed request structs.
- **Command run flow**: `cmd/terraci/cmd` parses cobra flags, calls `runflow.Prepare`, then passes `runflow.Prepared` into a typed command flow under `cmd/terraci/internal/*flow`; command files own only output/log presentation and file/stdout writes.
- **Command/usecase boundary**: command callbacks use `plugin.CommandPlugin[T]` and `plugin.RequireEnabled`, parse flags into request structs, call a usecase, then handle artifact persistence and output explicitly.
- **PipelineContributor(ctx)**: plugins add standalone DAG jobs through `pipeline.NewPluginCommandJob` + `pipeline.NewContribution`, return builder errors, and use `PipelineContributionGate` for optional jobs; `nil, nil` is invalid
- **ServiceDir**: configurable project directory; `AppContext.ServiceDir` (absolute) for runtime, `AppContext.Config().ServiceDir()` (relative) for pipeline templates
- **Immutable config boundary**: `Config.Clone()` and `config.Snapshot` own deep-copy semantics. `AppContext` stores a snapshot; production plugin code reads through accessors and leaves `MutableCopy()` to tests or explicit compatibility adapters.
- **Command boundary**: plugin command callbacks use `plugin.CommandPlugin[T](cmd, name)` and `plugin.RequireEnabled(...)`; low-level cobra context binding is framework-owned. Command binding and disabled-plugin failures are typed errors.
- **SDK contract kit**: plugin SDK behavior is tested through `pkg/plugin/plugintest`; CI/report behavior is tested through `pkg/ci/citest`. New plugins should copy these contract helpers for config immutability, command binding, runtime creation, contributions, lifecycle, init wizard, providers, change detection, rendered reports, and artifact lifecycle.
- **Init wizard flow**: command code owns cobra, TTY checks, huh rendering, YAML preview, and file writes. `cmd/terraci/internal/initflow` owns defaults, contributor collection, display group ordering/merge rules, duplicate extension detection, and final config assembly.
- **Init extension contracts**: init wizard plugins define package-local `initwiz.StateKey[T]` values, build fields through `initwiz.NewStringField` / `NewBoolField` / `NewSelectField`, and return typed config structs/maps through `initwiz.NewInitContribution`. Core owns YAML node encoding and defensive copies; initflow owns duplicate detection and final assembly. Do not return loose extension maps from plugin init code.
- **Report artifact lifecycle**: plan-aware producers use `PlanResultCollection -> ci.ArtifactRun -> ci.NewRenderedReport -> ci.PublishArtifacts(...)`. `PublishArtifacts` always persists raw results and removes stale reports on nil/build errors. Report-only producers may use `SaveReport`.
- **Report sections via typed render payloads**: producer plugins call `ci.NewRenderedReport(...)` and publish only validated `ci.ReportSectionKindRendered` sections with constructor-built `ci.RenderBlock` / `ci.RenderValue` payloads and the current rendered payload `schema_version`. `ReportSection`, `RenderBlock`, `RenderTable`, and `RenderValue` internals are private; use constructors/getters plus `ci.DecodeRenderSection`, not raw payload access. Summary/local renderers consume the generic render model through `plugins/internal/reportrender` and stay unaware of cost/policy/tfupdate domain structs.
- **Report freshness**: `pkg/ci.SelectCurrentReports` owns current/stale/degraded policy. Summary and localexec skip reports whose non-empty `plan_results_fingerprint` does not match the current plan collection. Missing provenance is accepted as degraded mode.
- **Cost resource attributes**: cost estimation uses `Terraform plan map -> resourcedef.RawAttrs -> resourcespec.TypedSpec parser -> resourcedef.Attributes -> resolver`. Raw Terraform `map[string]any` is allowed only at plan ingestion, test fixtures, and `RawAttrs` construction; resource definitions and runtime callbacks consume parsed typed attributes.
- **Policy OPA boundary**: policy checks use `plan.json bytes -> policyinput.PlanDocument -> policyinput.Envelope -> OPA adapter -> typed Evaluation -> Result/Summary -> report/output`. Raw OPA/JSON maps are allowed only inside the input document, OPA adapter, typed metadata value object, and test fixtures; use-cases and reports consume typed policy values.
- **Zero cross-plugin imports**: plugins communicate only via `pkg/plugin` capability helpers, shared `pkg/ci` types, and `ci.ReportStore` artifacts
- **Shared workflow**: `workflow.PlanProject()` is the high-level canonical project planning API for built-in production code: scan, filter, parse, graph building, optional target selection, changed-only, and library diagnostics. Lower-level scan/filter/target helpers are package-private internals inside `pkg/workflow`. `workflow.ChangeDetector`, `workflow.ChangeDetectionRequest`, and `workflow.ChangeDetectionResult` are plugin-agnostic; `plugin.ChangeDetectionProvider` embeds that workflow contract plus `plugin.Plugin`.
- **Localexec boundary**: keep shell/tfexec details inside `plugins/localexec`; `pkg/execution` stays provider-agnostic scheduler/executor infrastructure that consumes `*pipeline.IR` and emits value-based execution events/results. `localexec/internal/flow` returns a typed result and never imports render/output packages; CLI code injects progress event sinks and handles final rendering.
- **Reference runtime-heavy plugins**: `cost`, `policy`, `tfupdate`
- **Parser architecture**: keep `pkg/parser` as a thin public facade; put orchestration, extraction, resolution, and source mechanics in `pkg/parser/internal/*` around the shared `pkg/parser/model`
- **Tfupdate architecture**: keep `plugins/tfupdate` command/runtime surfaces thin; engine internals live under `planner`, `lockfile`, `sourceaddr`, `registrymeta`, `usecase`, `registryclient`, `tffile`, and `tfwrite`
- **Performance priority**: for `terraci tfupdate`, optimize registry lookup reuse and solver throughput before micro-optimizing formatting/output; parser hot paths matter because `tfupdate` rides on them transitively

## CLI Commands

```bash
terraci generate -o .gitlab-ci.yml          # Generate pipeline
terraci generate --changed-only             # Only changed modules
terraci generate --plan-only                # Plan jobs only
terraci validate                            # Validate config
terraci graph --format dot --stats          # Dependency graph
terraci init                                # Interactive wizard
terraci init --ci --provider gitlab         # Non-interactive
terraci cost                                # Cloud cost estimation
terraci summary                             # Post MR/PR comment
terraci local-exec plan                     # Local plan DAG
terraci local-exec run --changed-only       # Full local execution flow for changed modules
terraci policy pull && terraci policy check # Policy checks
terraci tfupdate                            # Terraform dependency resolution and lock synchronization
terraci schema                              # JSON schema
terraci version                             # Version + plugin info

xterraci build                              # Build with all plugins
xterraci build --without cost               # Exclude plugin
xterraci build --with github.com/x/plugin   # Add external plugin
xterraci list-plugins                       # Show available plugins
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/hashicorp/hcl/v2` | HCL parsing |
| `github.com/zclconf/go-cty` | CTY types for HCL |
| `github.com/hashicorp/terraform-json` | Terraform plan JSON types |
| `go.yaml.in/yaml/v4` | YAML serialization |
| `gitlab.com/gitlab-org/api/client-go` | GitLab API client |
| `github.com/google/go-github/v68` | GitHub API client |
| `github.com/open-policy-agent/opa` | Embedded OPA engine |
| `github.com/go-git/go-git/v6` | Git operations |
| `oras.land/oras-go/v2` | OCI registry operations |
| `github.com/invopop/jsonschema` | JSON schema generation |
| `github.com/caarlos0/log` | Structured logging |
| `charm.land/bubbletea/v2` | TUI framework |
| `charm.land/huh/v2` | TUI form components |
| `charm.land/lipgloss/v2` | TUI styling |
| `golang.org/x/sync` | Concurrency utilities |
