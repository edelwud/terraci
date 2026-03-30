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
    ├── generate.go             # Pipeline generation (uses plugin.ResolveProvider())
    ├── graph.go                # Dependency graph visualization
    ├── validate.go             # Config/project validation
    ├── filters.go              # filterFlags struct — shared filter flags, mergedFilterOpts()
    ├── init.go                 # Config initialization (--ci mode), initStateDefaults()
    ├── init_tui.go             # Interactive TUI wizard, dynamic plugin groups
    ├── schema.go               # JSON schema (includes plugin schemas)
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

pkg/                            # Public API — importable by external plugins
├── plugin/
│   ├── plugin.go               # Plugin interface + capability interfaces (split CI provider)
│   ├── base.go                 # BasePlugin[C] generic embedding
│   ├── enable.go               # EnablePolicy enum
│   ├── registry.go             # Register(), All(), ByCapability[T](), ResolveProvider()
│   ├── context.go              # AppContext (with ServiceDir, Reports, Freeze)
│   ├── runtime.go              # RuntimeProvider + RuntimeAs() for lazy plugin runtimes
│   ├── reports.go              # ReportRegistry — in-memory report exchange
│   ├── init_state.go           # StateMap — typed form state with pointer getters for huh
│   ├── helpers.go              # CollectContributions() with framework-level filtering
│   └── plugintest/             # Shared plugin-facing test helpers
├── ciprovider/
│   └── types.go                # Shared CI types: Image, MRCommentConfig
├── pipeline/
│   ├── types.go                # IR, Level, ModuleJobs, Job, Step, Phase, Contribution, ContributedJob
│   ├── builder.go              # Build(opts) — constructs provider-agnostic pipeline IR
│   ├── pipeline.go             # Generator, GeneratedPipeline interfaces
│   ├── common.go               # JobPlan, BuildJobPlan, JobName, ResolveDependencyNames
│   ├── env.go                  # BuildModuleEnvVars
│   └── scripts.go              # ScriptConfig, PlanScript, ApplyScript
├── config/
│   ├── config.go               # Config (service_dir, structure, exclude, include, plugins map)
│   ├── builder.go              # BuildConfigFromPlugins(), SetPluginValue()
│   ├── pattern.go              # ParsePattern, PatternSegments
│   └── schema.go               # GenerateJSONSchema (with plugin schemas)
├── ci/                         # Provider-agnostic CI types, Report, CommentService
├── discovery/                  # Module, Scanner, ModuleIndex, PlanScanner
├── parser/                     # HCL parser, DependencyExtractor
├── graph/                      # DependencyGraph, algorithms, visualization
├── filter/                     # GlobFilter, flags
├── workflow/                   # Module discovery, filtering, graph building
├── errors/                     # Typed errors
└── log/                        # Structured logging

plugins/                        # Built-in plugins — one file per capability
├── gitlab/
│   ├── plugin.go               # init, BasePlugin[*Config] embed, FlagOverridable
│   ├── lifecycle.go            # Preflightable (cheap MR context detection)
│   ├── generator.go            # EnvDetector + CIMetadata + GeneratorFactory + CommentFactory
│   ├── init_wizard.go          # InitContributor
│   └── internal/               # (package gitlabci) config, client, generator, MR service, types
├── github/
│   ├── plugin.go               # init, BasePlugin[*Config] embed, FlagOverridable
│   ├── lifecycle.go            # Preflightable (cheap PR context detection)
│   ├── generator.go            # EnvDetector + CIMetadata + GeneratorFactory + CommentFactory
│   ├── init_wizard.go          # InitContributor
│   └── internal/               # (package githubci) config, client, generator, PR service, types
├── cost/
│   ├── plugin.go               # init, BasePlugin[*CostConfig] embed
│   ├── lifecycle.go            # Preflightable (cheap config/cache validation)
│   ├── commands.go             # CommandProvider (terraci cost)
│   ├── runtime.go              # RuntimeProvider (lazy estimator construction)
│   ├── usecases.go             # Discovery/estimate/artifact orchestration
│   ├── pipeline.go             # PipelineContributor
│   ├── init_wizard.go          # InitContributor
│   ├── output.go               # CLI rendering helpers
│   ├── report.go               # CI report assembly
│   └── internal/               # (package costengine) — layered cost estimation engine
│       ├── engine/             #   Estimation orchestration, Terraform adapter, prefetch/execution
│       ├── runtime/            #   Provider catalog, pricing runtime, resolver, prefetch service
│       ├── model/              #   Cost result/config types + tree/module helpers
│       ├── results/            #   Result assembly layer
│       ├── cloud/              #   Cloud provider registry (init() + RegisterCloudProvider)
│       │   ├── registry.go     #     Provider definitions + global registry
│       │   ├── aws/            #     AWS provider + handler subpackages
│       │   │   ├── provider.go #       init() self-registration + provider definition
│       │   │   ├── ec2/, rds/, elb/, eks/, elasticache/, serverless/, storage/
│       │   └── awskit/         #     AWS utilities (no handler imports)
│       │       ├── runtime.go, standard_lookup.go, describe.go, services.go
│       ├── handler/            #   Provider-agnostic handler interfaces
│       │   ├── handler.go      #     ResourceHandler + capability interfaces
│       │   ├── registry.go     #     Provider-scoped handler registry
│       │   ├── attrs.go, calc.go
│       ├── pricing/            #   Disk-based pricing cache + types
│       ├── handlertest/        #   Handler contract test kit
│       ├── runtimetest/        #   Runtime contract test kit
│       └── enginetest/         #   Engine fixture/test helpers
├── policy/
│   ├── plugin.go               # init, BasePlugin[*Config] embed
│   ├── lifecycle.go            # Preflightable (OPA/source validation)
│   ├── commands.go             # CommandProvider (terraci policy pull/check)
│   ├── runtime.go              # RuntimeProvider (lazy puller/runtime construction)
│   ├── usecases.go             # Pull/check orchestration
│   ├── pipeline.go             # PipelineContributor (policy-check job)
│   ├── version.go              # VersionProvider (OPA version)
│   ├── init_wizard.go          # InitContributor
│   ├── output.go               # CLI rendering
│   ├── report.go               # CI report assembly
│   └── internal/               # (package policyengine) OPA engine, checker, sources
├── update/
│   ├── plugin.go               # init, BasePlugin[*Config] embed
│   ├── lifecycle.go            # Preflightable (cheap config validation)
│   ├── commands.go             # CommandProvider (terraci update)
│   ├── runtime.go              # RuntimeProvider (lazy registry/runtime construction)
│   ├── usecases.go             # Update-check orchestration
│   ├── output.go               # CLI rendering
│   ├── report.go               # CI report assembly
│   ├── init_wizard.go          # InitContributor
│   └── internal/               # (package updateengine) parser, checker, registry client
├── summary/
│   ├── plugin.go               # init, BasePlugin[*Config] embed
│   ├── commands.go             # CommandProvider (terraci summary)
│   ├── pipeline.go             # PipelineContributor (PhaseFinalize summary job)
│   ├── init_wizard.go          # InitContributor
│   ├── output.go               # CLI output helpers
│   └── internal/               # (package summaryengine) config, renderer, report_loader
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

Compile-time plugins via `init()` + blank import (Caddy/database-sql pattern). Plugins register via `plugin.Register()`, core discovers via `plugin.ByCapability[T]()`.

### Plugin File Convention

Each plugin follows one-file-per-capability, with runtime-heavy plugins also using a lazy runtime layer:
- `plugin.go` — init(), Plugin struct with BasePlugin[C] embedding, Reset(), FlagOverridable
- `lifecycle.go` — Preflightable
- `runtime.go` — RuntimeProvider for lazy runtime construction
- `usecases.go` — command orchestration over typed runtime
- `commands.go` — CommandProvider with cobra definitions
- `generator.go` — EnvDetector + CIMetadata + GeneratorFactory + CommentFactory
- `pipeline.go` — PipelineContributor(ctx) (no self-check, framework filters)
- `init_wizard.go` — InitContributor (uses typed *StateMap)
- `version.go` — VersionProvider
- `output.go` — Rendering/formatting helpers
- `report.go` — CI report assembly
- `detect.go` — ChangeDetectionProvider

### Plugin Lifecycle

```
1. Register    — init() calls plugin.Register() with BasePlugin[C] embedding
2. Configure   — ConfigLoader.DecodeAndSet() for plugins with config in .terraci.yaml
3. Preflight   — Preflightable.Preflight() performs cheap validation/env detection
4. Freeze      — AppContext.Freeze() prevents further mutations
5. Execute     — Commands/use-cases lazily build RuntimeProvider runtimes as needed
```

### Capability Interfaces

| Interface | Purpose | Implemented by |
|-----------|---------|----------------|
| `Plugin` | Base: Name(), Description() | all |
| `ConfigLoader` | Config section under `plugins:` + IsEnabled() via EnablePolicy | gitlab, github, cost, policy, summary, update |
| `CommandProvider` | CLI subcommands | cost, policy, summary, update |
| `Preflightable` | Cheap startup validation / env detection | gitlab, github, cost, policy, git, update |
| `RuntimeProvider` | Lazy command-time runtime construction | cost, policy, update |
| `EnvDetector` | CI environment detection | gitlab, github |
| `CIMetadata` | Provider name, pipeline ID, commit SHA | gitlab, github |
| `GeneratorFactory` | Pipeline generator creation | gitlab, github |
| `CommentFactory` | MR/PR comment service creation | gitlab, github |
| `FlagOverridable` | Direct CLI flag overrides (--plan-only, --auto-approve) | gitlab, github |
| `VersionProvider` | Version info contributions | policy |
| `ChangeDetectionProvider` | VCS change detection | git |
| `InitContributor` | Init wizard form fields + config building | gitlab, github, cost, policy, summary, update |
| `PipelineContributor` | Pipeline steps/jobs via Contribution | cost, policy, summary |

### BasePlugin[C] Generic Embedding

Plugins with config embed `BasePlugin[C]` which auto-implements:
- `Name()`, `Description()`, `ConfigKey()`, `NewConfig()`, `DecodeAndSet()`, `IsConfigured()`, `IsEnabled()`, `Config()`, `Reset()`
- `EnablePolicy` controls enabled semantics: `EnabledWhenConfigured` (gitlab/github), `EnabledExplicitly` (cost/policy), `EnabledByDefault` (summary), `EnabledAlways` (git)

### Shared Types

`pkg/ciprovider/` contains types shared by CI providers: `Image` (with YAML shorthand), `MRCommentConfig`. Both gitlab and github internal packages use type aliases to these.

### Pipeline IR

`pkg/pipeline.Build(opts)` creates a provider-agnostic IR. Generators transform it to YAML:

```
pipeline.Build(opts) → IR{Levels, Jobs}
  ↓
GitLab: IR → Pipeline{Stages, Jobs} → YAML
GitHub: IR → Workflow{Jobs, Steps} → YAML
```

Plugins contribute via `PipelineContributor.PipelineContribution(ctx)`:
- `Contribution.Steps` — injected into plan/apply jobs (PrePlan/PostPlan/PreApply/PostApply)
- `Contribution.Jobs` — standalone jobs (e.g., policy-check after plans)

### Provider Resolution

`plugin.ResolveProvider()` returns `*CIProvider` (struct wrapping EnvDetector + CIMetadata + GeneratorFactory + CommentFactory): CI env → `TERRACI_PROVIDER` env → single registered → IsConfigured() filter → error. Core has zero knowledge of specific providers. Commands that don't need config use `Annotations["skipConfig"]` to skip config loading in `PersistentPreRunE`. CLI flag overrides use `FlagOverridable` for direct struct mutation (no encode-decode cycle).

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

plugins:
  gitlab:
    image: { name: hashicorp/terraform:1.6 }
    terraform_binary: terraform
    plan_enabled: true
    auto_approve: false
    mr:
      comment: { enabled: true }
      summary_job:
        image: { name: "ghcr.io/edelwud/terraci:latest" }

  # cost:
  #   enabled: true
  #   cache_dir: ~/.terraci/pricing
  #   cache_ttl: "24h"

  # policy:
  #   enabled: true
  #   sources: [{ path: terraform }]
  #   on_failure: block
```

Core config: `service_dir`, `structure`, `exclude`, `include`, `library_modules`, `plugins` (opaque map). All provider/feature config under `plugins:`.

## Data Flow

### Generate pipeline
1. `workflow.Run(ctx, opts)` — scan → filter → parse → graph
2. `ChangeDetectionProvider.DetectChangedModules()` (if --changed-only)
3. `plugin.CollectContributions()` — gather PipelineContributor steps/jobs
4. `pipeline.Build(opts)` — construct provider-agnostic IR
5. `GeneratorProvider.NewGenerator()` — transform IR to provider YAML

### Summary
1. `discovery.ScanPlanResults()` → PlanResultCollection
2. Load plugin reports from `{serviceDir}/*-report.json` (file-based enrichment)
3. `summaryengine.EnrichPlans()` merges report data into plan results
4. `summaryengine.ComposeComment()` renders markdown
5. `plugin.ResolveProvider()` → `NewCommentService()` → `UpsertComment(ctx, body)`

### Init wizard
1. `initStateDefaults()` populates shared defaults (provider, binary, pattern, plan_enabled)
2. Core groups: Basics, Structure, Pipeline Options
3. `InitContributor` plugins add dynamic form groups
4. `BuildConfigFromPlugins(pattern, pluginConfigs)` assembles config (returns `(*Config, error)`)

## Key Patterns

- **Plugin-first**: core is provider-agnostic; all logic in `plugins/`
- **One file per capability**: plugin.go < 30 lines; each interface in its own file
- **Compile-time extensibility**: `xterraci build --with/--without` for custom binaries
- **Pipeline IR**: `pkg/pipeline.Build()` → provider transforms to YAML
- **Preflight, then lazy runtime**: framework performs cheap startup validation; heavy plugin state is built lazily inside RuntimeProvider/use-cases
- **PipelineContributor(ctx)**: plugins inject steps/jobs without cross-plugin imports or cached service-dir state
- **ServiceDir**: configurable project directory; `AppContext.ServiceDir` (absolute) for runtime, `Config.ServiceDir` (relative) for pipeline templates
- **File-based reports**: plugins write `{serviceDir}/{plugin}-report.json`; summary plugin loads and merges them
- **Zero cross-plugin imports**: plugins communicate only via registry + shared types + file-based reports
- **Shared workflow**: `workflow.Run()` — scan, filter, parse, graph building
- **Reference runtime-heavy plugins**: `cost`, `policy`, `update`

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
terraci policy pull && terraci policy check # Policy checks
terraci update                              # Dependency version checks
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
