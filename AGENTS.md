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
    ├── app.go                  # App struct, PluginContext() with ServiceDir, InitPluginConfigs()
    ├── root.go                 # NewRootCmd(), plugin lifecycle (Init), dynamic commands
    ├── generate.go             # Pipeline generation (builds IR, calls provider.NewGenerator(ctx, ir))
    ├── graph.go                # Dependency graph visualization
    ├── validate.go             # Config/project validation
    ├── filters.go              # filterFlags struct — shared filter flags, mergedFilterOpts()
    ├── init.go                 # Config initialization (--ci mode), initStateDefaults()
    ├── init_tui.go             # Interactive TUI wizard, dynamic plugin groups
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

pkg/                            # Public API — importable by external plugins (plugin-agnostic core + plugin SDK)
├── plugin/                     # Core plugin SDK — interfaces, BasePlugin, AppContext
│   ├── plugin.go               # Plugin
│   ├── lifecycle.go            # ConfigLoader, Preflightable
│   ├── commands.go             # CommandProvider, FlagOverridable, VersionProvider
│   ├── ci_provider.go          # EnvDetector, CIInfoProvider, PipelineGeneratorFactory, CommentServiceFactory, ResolvedCIProvider
│   ├── cache.go                # KVCacheProvider, BlobStoreProvider (single NewBlobStore with options)
│   ├── change.go               # ChangeDetectionProvider
│   ├── contribution.go         # PipelineContributor
│   ├── base.go                 # BasePlugin[C] generic embedding
│   ├── enable.go               # EnablePolicy enum
│   ├── context.go              # AppContext + AppContextOptions constructor
│   ├── context_binding.go      # AppContext.Update / SetResolver / BeginCommand / Freeze
│   ├── command_binding.go      # CommandInstance[T] — typed command-scoped lookup
│   ├── runtime.go              # RuntimeProvider + RuntimeAs() + BuildRuntime[T]()
│   ├── resolver.go             # Resolver interface (extended with capability resolution methods)
│   ├── noop_resolver.go        # default no-op Resolver (never nil)
│   ├── reports.go              # ReportRegistry — in-memory report exchange
│   ├── registry/               # Plugin factory catalog + per-command Registry
│   │   ├── registry.go         # RegisterFactory(), New(), Catalog, Registry implements plugin.Resolver
│   │   └── resolve.go          # Registry.ResolveCIProvider/ResolveChangeDetector/Resolve*Provider/CollectContributions/PreflightsForStartup
│   ├── initwiz/                # Init wizard state + types
│   │   ├── state.go            # StateMap — typed form state with pointer getters for huh
│   │   └── types.go            # InitContributor, InitGroupSpec, InitField, FieldType
│   └── plugintest/             # Shared plugin-facing test helpers + mock doubles + NoopResolver
├── pipeline/                   # Plugin-agnostic pipeline IR
│   ├── types.go                # IR, Level, ModuleJobs, Job, Operation, TerraformOperation, Contribution, ContributedJob
│   ├── builder.go              # Build(opts) — constructs provider-agnostic pipeline IR
│   ├── pipeline.go             # Generator interface (Generate()/DryRun() — IR bound at construction)
│   ├── common.go               # JobPlan, JobName, ResolveDependencyNames, IR.DryRun
│   ├── jobs.go                 # IR.JobRefs / PlanJobsForLevel / ApplyJobsForLevel
│   ├── env.go                  # ModuleEnvVars
│   ├── scripts.go              # ScriptConfig, NewPlanOperation, NewApplyOperation (IR construction only)
│   └── cishell/                # Shell renderer — keeps shell-specific knowledge out of the IR
│       └── render.go           # RenderOperation: pipeline.Operation → POSIX shell command lines
├── ci/                         # Plugin-agnostic CI types
│   ├── report.go, report_types.go, report_validation.go, section.go
│   │                           #   Report (Producer/Title/Status/Summary/Provenance/Sections), ReportSection (opaque Payload), EncodeSection/DecodeSection generics, OverviewSection/ModuleTableSection/FindingsSection/DependencyUpdatesSection payloads
│   ├── plan.go                 # PlanResult (canonical for both in-memory + persisted), PlanResultCollection, PlanStatus
│   ├── service.go              # CommentService
│   └── shared.go               # Image, CommentMarker
├── cache/blobcache/            # Blob store contract + cache layer (owns Store/Meta/Object/PutOptions/Info/Inspector/Describer/HealthChecker, Describe, Check, Cache, Policy)
├── config/
│   ├── types_config.go         # Config (service_dir, structure, exclude, include, library_modules, extensions map[string]yaml.Node)
│   ├── builder.go              # BuildConfig() — assembles Config from pattern/execution/extensions
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
├── execution/                  # Plugin-agnostic local execution
│   ├── executor.go             # Executor.Execute(ctx, *pipeline.IR) — IR is the single execution input
│   ├── scheduler.go            # DefaultScheduler.Schedule(*pipeline.IR) — barrier-grouped JobGroups
│   ├── results.go, worker_pool.go, workspace.go, config.go
├── graph/                      # DependencyGraph, algorithms, visualization
├── filter/                     # Public surface: Options + Apply + Flags + ParseSegmentFilters (concrete filter types unexported)
├── workflow/                   # Module discovery, filtering, graph building, target resolution. workflow.ChangeDetector = plugin.ChangeDetectionProvider alias
├── errors/                     # Typed errors (ConfigError, ScanError, ParseError, NoModulesError)
└── log/                        # Thin wrapper over caarlos0/log

plugins/                        # Built-in plugins — one file per capability
├── gitlab/
│   ├── plugin.go               # init, BasePlugin[*Config] embed, FlagOverridable
│   ├── lifecycle.go            # Preflightable (cheap MR context detection)
│   ├── generator.go            # EnvDetector + CIInfoProvider + PipelineGeneratorFactory(ctx, *pipeline.IR) + CommentServiceFactory
│   ├── init_wizard.go          # InitContributor
│   └── internal/               # config, generator, MR service, domain types
│       └── generate/
│           ├── buildir.go      # BuildPipelineIR helper for tests (IR construction with plugin settings)
│           └── generator.go    # Generator stores *pipeline.IR; Generate()/DryRun() with no args
├── github/
│   ├── plugin.go               # init, BasePlugin[*Config] embed, FlagOverridable
│   ├── lifecycle.go            # Preflightable (cheap PR context detection)
│   ├── generator.go            # EnvDetector + CIInfoProvider + PipelineGeneratorFactory(ctx, *pipeline.IR) + CommentServiceFactory
│   ├── init_wizard.go          # InitContributor
│   └── internal/generate/      # IR-bound generator + buildir.go test helper
├── cost/
│   ├── plugin.go               # init, BasePlugin[*CostConfig] embed
│   ├── lifecycle.go            # Preflightable (cheap config/cache validation)
│   ├── commands.go             # CommandProvider (terraci cost)
│   ├── runtime.go              # RuntimeProvider (lazy estimator construction)
│   ├── usecases.go             # Discovery/estimate/artifact orchestration
│   ├── pipeline.go             # PipelineContributor
│   ├── init_wizard.go          # InitContributor
│   ├── output.go               # CLI rendering helpers
│   ├── report.go               # CI report assembly via ci.MustEncodeSection
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
│   └── internal/               # (package summaryengine) config, renderer, report_loader
├── localexec/
│   ├── plugin.go               # init, Plugin struct
│   ├── commands.go             # CommandProvider (terraci local-exec with plan/run only)
│   ├── contract.go             # Public stable NewExecutor(...) boundary for in-process callers
│   └── internal/
│       ├── executor.go         # Thin adapter from public contract to internal flow
│       ├── flow/               # Use-case orchestration: workflow → targets → IR → execute → render
│       ├── planner/            # pipeline.Build → *pipeline.IR adapter with contribution filtering
│       ├── render/             # Progress output and local CLI rendering
│       ├── runner/             # Shell/Terraform runners + DAG job orchestration
│       ├── spec/               # Internal validated execute request/mode types
│       └── targeting/          # Shared workflow target-resolution adapter
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
└── git/
    ├── plugin.go               # init, Plugin struct (no config, no BasePlugin)
    ├── lifecycle.go            # Preflightable (cheap repo detection)
    ├── detect.go               # ChangeDetectionProvider
    └── internal/               # (package gitclient) client, detector, diff

internal/                       # Private — only terraform eval
└── terraform/
    ├── eval/                   # NewContext(), 30+ Terraform functions
    └── plan/                   # ParseJSON, ResourceChange, AttrDiff
```

## Plugin System

### Architecture

Compile-time plugins via `init()` + blank import (Caddy/database-sql pattern). Plugins register factories via `registry.RegisterFactory()`, core creates a fresh `*registry.Registry` for each command run; that `*Registry` directly implements `plugin.Resolver`. Capability discovery uses `registry.ByCapabilityFrom[T](resolver)`. Core types (interfaces, BasePlugin, AppContext) live in `pkg/plugin`; plugin catalog and per-command registries live in `pkg/plugin/registry`; init wizard types in `pkg/plugin/initwiz`.

The core `pkg/` tree is **plugin-agnostic** — no package outside `pkg/plugin` imports the plugin SDK. Plugin extensibility hangs entirely off `pkg/plugin`'s capability interfaces.

### Plugin File Convention

Each feature/plugin follows one-file-per-capability where it applies, with runtime-heavy plugins also using a lazy runtime layer. Backend plugins such as `diskblob` and `inmemcache` are intentionally smaller and only implement their relevant provider interfaces:
- `plugin.go` — init(), Plugin struct with BasePlugin[C] embedding, FlagOverridable
- `lifecycle.go` — Preflightable
- `runtime.go` — RuntimeProvider for lazy runtime construction
- `usecases.go` — command orchestration over typed runtime
- `commands.go` — CommandProvider with cobra definitions
- `generator.go` — EnvDetector + CIInfoProvider + PipelineGeneratorFactory + CommentServiceFactory
- `pipeline.go` — PipelineContributor(ctx) (no self-check, framework filters)
- `init_wizard.go` — initwiz.InitContributor (uses typed *initwiz.StateMap)
- `version.go` — VersionProvider
- `output.go` — Rendering/formatting helpers
- `report.go` — CI report assembly
- `detect.go` — ChangeDetectionProvider

### Plugin Lifecycle

```
1. Register    — init() calls registry.RegisterFactory() with a Plugin factory
2. Configure   — ConfigLoader.DecodeAndSet() for plugins with a config section under extensions:
3. Preflight   — Preflightable.Preflight() performs cheap validation/env detection
4. Freeze      — AppContext.Freeze() prevents further mutations
5. Execute     — Commands/use-cases lazily build RuntimeProvider runtimes as needed
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
| `FlagOverridable` | Direct CLI flag overrides (--plan-only, --auto-approve) | gitlab, github |
| `VersionProvider` | Version info contributions | policy |
| `ChangeDetectionProvider` | VCS change detection | git |
| `KVCacheProvider` | Named key/value cache backend resolution | inmemcache |
| `BlobStoreProvider` | Named blob/object store backend (`NewBlobStore(ctx, appCtx, opts)`) | diskblob |
| `InitContributor` | Init wizard form fields + config building | gitlab, github, cost, policy, summary, tfupdate |
| `PipelineContributor` | Pipeline steps/jobs via Contribution | cost, policy, summary, tfupdate |

### BasePlugin[C] Generic Embedding

Plugins with config embed `BasePlugin[C]` which auto-implements:
- `Name()`, `Description()`, `ConfigKey()`, `NewConfig()`, `DecodeAndSet()`, `IsConfigured()`, `IsEnabled()`, `Config()`, `Reset()`
- `EnablePolicy` controls enabled semantics: `EnabledWhenConfigured` (gitlab/github), `EnabledExplicitly` (cost/policy/tfupdate), `EnabledByDefault` (summary/diskblob/inmemcache), `EnabledAlways` (git)

### AppContext

Construction goes through an options struct — `plugin.NewAppContext(plugin.AppContextOptions{Config, WorkDir, ServiceDir, Version, Reports, Resolver})` — every field is optional. `ctx.Resolver()` is **never nil**: when no resolver is bound, a no-op resolver returns sentinel errors so plugins can call `ctx.Resolver().ResolveCIProvider()` etc. without nil-checks.

`AppContext.Config()` returns the bound `*config.Config` pointer directly; plugins must treat it as read-only.

### Resolver

The single `plugin.Resolver` interface combines lookup (`All`, `GetPlugin`) with capability resolution (`ResolveCIProvider`, `ResolveChangeDetector`, `ResolveKVCacheProvider`, `ResolveBlobStoreProvider`, `CollectContributions`, `PreflightsForStartup`). `*registry.Registry` is the production implementation; `plugintest.NoopResolver` is the test default-deny.

### Shared Types

`pkg/ci/` contains shared CI-domain types including provider-shared config such as `Image` (with YAML shorthand). `ci.Report` is the typed file-based report contract shared by cost/policy/tfupdate/summary; reports carry optional provenance metadata for local validation. Both gitlab and github internal packages use type aliases to these.

`ci.ReportSection` is a neutral envelope with an opaque `Payload json.RawMessage`. Producers call `ci.MustEncodeSection(kind, title, summary, status, body)` with a typed body (`ci.OverviewSection`, `ci.FindingsSection`, etc.); consumers call `ci.DecodeSection[T](section)`.

`ci.PlanResult` is the canonical representation of one module's plan outcome — used both in-memory and on disk; `ci.PlanResultCollection` aggregates them with a stable fingerprint.

### Pipeline IR

`pkg/pipeline.Build(opts)` creates a provider-agnostic IR. Generators transform it to YAML:

```
pipeline.Build(opts) → IR{Levels, Jobs}
  ↓
GitLab: IR → Pipeline{Stages, Jobs} → YAML
GitHub: IR → Workflow{Jobs, Steps} → YAML
```

The IR is the **single source** for downstream consumers — `pipeline.Generator` is constructed with an `*IR` and `Generate() (GeneratedPipeline, error)` / `DryRun() (*DryRunResult, error)` take no further arguments. Job-level access methods live on `*IR`: `JobRefs()`, `PlanJobsForLevel(idx)`, `ApplyJobsForLevel(idx)`, plus `JobNames()` / `AllPlanNames()` / `ContributedJobNames()`.

Shell rendering (`cd module && ${TERRAFORM_BINARY} init && plan -out=…`) lives in `pkg/pipeline/cishell` (`cishell.RenderOperation(op)`) — never in the IR package itself. Providers driving Terraform via tfexec instead of shell don't need cishell.

Plugins contribute via `PipelineContributor.PipelineContribution(ctx)`:
- `Contribution.Jobs` — standalone DAG jobs with typed resource inputs/outputs

### Provider Resolution

`Registry.ResolveCIProvider()` returns `*plugin.ResolvedCIProvider` (struct wrapping EnvDetector + CIInfoProvider + PipelineGeneratorFactory + CommentServiceFactory): `TERRACI_PROVIDER` env → CI env detection → single active provider → error. Core has zero knowledge of specific providers. Commands that don't need config use `Annotations["skipConfig"]` to skip config loading in `PersistentPreRunE`. CLI flag overrides use `FlagOverridable` for direct struct mutation (no encode-decode cycle).

### Service Directory

`AppContext.ServiceDir` — resolved absolute path to project service directory for runtime file I/O. Configurable via `service_dir` in config (default `.terraci`). For pipeline artifact paths (CI templates), use `Config.ServiceDir` which preserves the original relative value.

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
  plan_enabled: true

extensions:
  gitlab:
    image: { name: hashicorp/terraform:1.6 }
    cache_enabled: true
    cache:
      policy: pull-push
      paths: [ "{module_path}/.terraform/" ]

  summary:
    on_changes_only: false

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
1. `workflow.Run(ctx, opts)` — scan → filter → parse → graph
2. `ChangeDetectionProvider.DetectChangedModules()` (if --changed-only)
3. `app.Plugins.CollectContributions(appCtx)` — gather a command-scoped snapshot of PipelineContributor jobs
4. `pipeline.Build(opts)` — construct provider-agnostic IR
5. `provider.NewGenerator(appCtx, ir)` — bind IR to provider; `generator.Generate()` writes YAML

### Summary
1. `planresults.Scan()` → PlanResultCollection
2. Load reports from `{serviceDir}/*-report.json` (file-based enrichment; filename uses `report.Producer`)
3. `summaryengine.EnrichPlans()` merges report data into plan results
4. `summaryengine.ComposeComment()` renders markdown
5. `appCtx.Resolver().ResolveCIProvider()` → `NewCommentService()` → `UpsertComment(ctx, body)`

### Local Execution
1. `workflow.Run(ctx, workflow.Options)` builds the canonical filtered module/graph result
2. `workflow.ResolveTargets(...)` applies merged filters, `--module`, `--changed-only`, and affected-library expansion
3. `localexec/internal/planner` calls `pipeline.Build(...)` to produce an `*pipeline.IR`
4. `pkg/execution.Executor.Execute(ctx, ir)` schedules jobs with dependency-aware DAG grouping
5. `localexec/internal/runner` executes shell/tfexec jobs locally
6. `localexec/internal/render` prints the execution DAG/job summary

### Init wizard
1. `initStateDefaults()` populates shared defaults (provider, binary, pattern, plan_enabled)
2. Core groups: Basics, Structure, Pipeline Options
3. `initwiz.InitContributor` plugins add dynamic form groups
4. `config.BuildConfig(pattern, execution, extensionConfigs)` assembles config (returns `(*Config, error)`)

## Key Patterns

- **Plugin-agnostic core**: nothing under `pkg/` (except `pkg/plugin/...`) imports the plugin SDK. No mention of "plugin" in core package types or YAML keys.
- **Plugin-first feature surfaces**: every CI provider, cost backend, policy engine, etc. lives in `plugins/`.
- **One file per capability**: plugin.go < 30 lines; each interface in its own file
- **Compile-time extensibility**: `xterraci build --with/--without` for custom binaries
- **Pipeline IR**: `pkg/pipeline.Build()` → provider transforms to YAML. The IR is the single execution input — generators and the local executor both consume `*pipeline.IR` directly.
- **IR-bound generators**: `PipelineGeneratorFactory.NewGenerator(ctx, *pipeline.IR)` — providers don't reach for depGraph/modules/contributions; the IR already encodes them.
- **Shell rendering separated from IR**: `pkg/pipeline/cishell.RenderOperation(op)` for shell-driven CI; the IR carries `pipeline.TerraformOperation` data only.
- **Canonical dry-run source**: dry-run stage/job counts derive from `*IR.DryRun(totalModules)`.
- **Preflight, then lazy runtime**: framework performs cheap startup validation; heavy plugin state is built lazily inside RuntimeProvider/use-cases
- **PipelineContributor(ctx)**: plugins add standalone DAG jobs without cross-plugin imports or cached service-dir state
- **ServiceDir**: configurable project directory; `AppContext.ServiceDir` (absolute) for runtime, `Config.ServiceDir` (relative) for pipeline templates
- **File-based reports**: producers write `{serviceDir}/{producer}-report.json` (e.g. `cost-report.json`); summary consumes plan/report files and posts comments but does not publish a pipeline resource
- **Report sections via opaque payload**: `ci.ReportSection.Payload` is `json.RawMessage`; producers use `ci.MustEncodeSection`, consumers use `ci.DecodeSection[T]`. Adding a new section kind requires no `pkg/ci` change.
- **Report provenance**: persisted reports may carry producer/run provenance; local consumers should validate provenance/fingerprint when correctness depends on current workspace artifacts
- **Zero cross-plugin imports**: plugins communicate only via `pkg/plugin` capability helpers + shared types + file-based reports
- **Shared workflow**: `workflow.Run()` — scan, filter, parse, graph building. `workflow.ChangeDetector` is an alias of `plugin.ChangeDetectionProvider`.
- **Localexec boundary**: keep shell/tfexec details inside `plugins/localexec`; `pkg/execution` stays provider-agnostic scheduler/executor infrastructure that consumes a raw `*pipeline.IR`.
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
