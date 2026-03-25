# TerraCi

CLI tool for analyzing Terraform projects, building dependency graphs, generating CI pipelines, and estimating AWS costs. Extended via compile-time plugin system.

## Build & Test

```bash
make build      # Build binary → build/terraci
make test       # Run tests with coverage
make test-short # Short tests
make lint       # golangci-lint or go vet
make fmt        # Format code
make install    # Install to $GOPATH/bin
```

## Project Structure

```
cmd/terraci/
├── main.go                     # Entry point — blank-imports all built-in plugins
└── cmd/
    ├── app.go                  # App struct, PluginContext(), InitPluginConfigs()
    ├── root.go                 # NewRootCmd(), PersistentPreRunE (config + plugin lifecycle), dynamic command registration
    ├── generate.go             # Pipeline generation (uses plugin.ResolveProvider())
    ├── filters.go              # filterFlags struct — shared filter flags and helpers
    ├── validate.go             # Config/project validation
    ├── graph.go                # Dependency graph visualization
    ├── init.go                 # Config initialization (entry point, --ci mode)
    ├── init_config.go          # buildConfigFromState() — shared between CLI/TUI
    ├── init_tui.go             # Interactive TUI wizard (bubbletea/huh), dynamic plugin groups
    ├── summary.go              # MR/PR comment posting via SummaryContributor plugins
    ├── schema.go               # JSON schema generation
    ├── completion.go           # Shell completion
    ├── man.go                  # Man page generation
    └── version.go              # Version info (uses VersionProvider plugins)

cmd/xterraci/
├── main.go                     # xterraci build CLI (like xcaddy)
├── builder.go                  # Build orchestration: temp dir, go mod, go build
├── codegen.go                  # Generates main.go with plugin imports
└── plugins.go                  # Built-in plugin import paths

pkg/                            # Public API — importable by external plugins
├── plugin/
│   ├── plugin.go               # Plugin interface + capability interfaces (see Plugin System below)
│   ├── registry.go             # Register(), All(), ByCapability[T](), ResolveProvider()
│   ├── hooks.go                # WorkflowHook, HookPhase, WorkflowState, CollectHooks(), RunHooks()
│   ├── context.go              # AppContext (lazy refresh), ExecutionContext (shared state), CommentSection
│   └── init_state.go           # StateMap — form state with pointer getters for huh binding
├── config/
│   ├── config.go               # Config (structure, exclude, include, library_modules, plugins map)
│   ├── builder.go              # BuildConfigFromPlugins(), SetPluginValue(), setPluginNode()
│   ├── pattern.go              # ParsePattern, PatternSegments
│   └── schema.go               # JSON schema generation
├── discovery/
│   ├── module.go               # Module struct (dynamic components + segments)
│   ├── scanner.go              # Scanner.Scan(ctx) — directory walk entry point
│   ├── collector.go            # moduleCollector — walk logic, context cancellation
│   ├── index.go                # ModuleIndex — fast lookups by ID/path/name
│   └── testing.go              # TestModule() helper for tests
├── parser/
│   ├── types.go                # Parser, ParsedModule, RemoteStateRef, BackendConfig, ModuleCall
│   ├── hcl.go                  # ParseModule(ctx), multi-pass locals evaluation, backend parsing
│   ├── resolve.go              # ResolveWorkspacePath, for_each resolution
│   └── dependency.go           # DependencyExtractor, backend index, ExtractAllDependencies(ctx)
├── graph/
│   ├── dependency.go           # DependencyGraph, Node, edges, traversal, library usage
│   ├── algorithms.go           # TopologicalSort, ExecutionLevels, DetectCycles
│   ├── affected.go             # GetAffectedModules, library changes, combined
│   ├── visualize.go            # ToDOT (clustered), ToPlantUML (nested groups)
│   └── stats.go                # GetStats (fan-in/fan-out, modules per level)
├── filter/
│   └── glob.go                 # GlobFilter, SegmentFilter, CompositeFilter, Apply()
├── ci/
│   ├── types.go                # ModulePlan, PlanResult, CommentData, PolicySummary
│   ├── comment.go              # CommentRenderer — shared PR/MR comment markdown
│   ├── helpers.go              # HasReportableChanges — shared on_changes_only logic
│   ├── plan_result.go          # ScanPlanResults, ParseModulePathComponents
│   └── service.go              # CommentService interface
├── pipeline/
│   ├── pipeline.go             # Generator and GeneratedPipeline interfaces
│   ├── env.go                  # BuildModuleEnvVars — shared TF_* env var builder
│   ├── common.go               # Shared plan/apply script builders
│   └── scripts.go              # Script template helpers
├── workflow/
│   └── module_workflow.go      # Run() with plugin hook execution at 5 phases
├── errors/
│   └── errors.go               # Typed errors: ConfigError, ScanError, ParseError, NoModulesError, etc.
└── log/
    └── log.go                  # Structured logging (wraps caarlos0/log)

plugins/                        # Built-in plugins — each has plugin.go + internal/
├── gitlab/
│   ├── plugin.go               # GeneratorProvider, ConfigProvider, Initializable, InitContributor
│   └── internal/               # (package gitlabci) config, client, generator, mr_service, types
├── github/
│   ├── plugin.go               # GeneratorProvider, ConfigProvider, Initializable, InitContributor
│   └── internal/               # (package githubci) config, client, generator, pr_service, types
├── cost/
│   ├── plugin.go               # SummaryContributor, CommandProvider, ConfigProvider, Initializable, InitContributor
│   └── internal/               # (package costengine) types, estimator, factory, tree, registry, aws/, pricing/
├── policy/
│   ├── plugin.go               # SummaryContributor, VersionProvider, CommandProvider, ConfigProvider, Initializable, InitContributor
│   └── internal/               # (package policyengine) config, checker, engine, result, source*
└── git/
    ├── plugin.go               # ChangeDetectionProvider, Initializable
    └── internal/               # (package gitclient) client, detector, diff

internal/                       # Private — only terraform eval stays here
└── terraform/
    ├── eval/
    │   ├── context.go          # NewContext() — path.module as abspath, SafeObjectVal
    │   └── functions.go        # 30+ Terraform functions
    └── plan/
        ├── types.go            # ParsedPlan, ResourceChange, AttrDiff
        ├── parser.go           # ParseJSON, countAction, buildAttrDiff
        └── maputil.go          # Nested map utilities
```

## Plugin System

### Architecture

Compile-time plugins using Go's `init()` + blank import pattern (like `database/sql` drivers, Caddy modules). Plugins register via `plugin.Register()` in `init()`, core discovers them via `plugin.ByCapability[T]()`.

### Plugin Lifecycle

```
1. Register    — init() calls plugin.Register() (triggered by blank import in main.go)
2. Configure   — ConfigProvider.SetConfig() receives decoded YAML from plugins: map
3. Initialize  — Initializable.Initialize() sets up resources (clients, caches, repo detection)
4. Execute     — Commands run, workflow hooks fire, SummaryContributor enriches data
5. Finalize    — Finalizable.Finalize() cleans up resources
```

### Capability Interfaces (`pkg/plugin/plugin.go`)

| Interface | Purpose | Implemented by |
|-----------|---------|----------------|
| `Plugin` | Base: Name(), Description() | all |
| `ConfigProvider` | Declares config section under `plugins:` | gitlab, github, cost, policy |
| `CommandProvider` | Adds CLI subcommands | cost (`terraci cost`), policy (`terraci policy`) |
| `GeneratorProvider` | CI pipeline generation + comment service | gitlab, github |
| `SummaryContributor` | Enriches plan results during `terraci summary` | cost, policy |
| `VersionProvider` | Contributes to `terraci version` output | policy (OPA version) |
| `ChangeDetectionProvider` | Detects changed modules from VCS | git |
| `InitContributor` | Contributes form fields + config to `terraci init` | gitlab, github, cost, policy |
| `FilterProvider` | Registers custom module filters | (available, unused) |
| `WorkflowHookProvider` | Injects behavior at workflow phases | (available, unused) |
| `Initializable` | Setup after config load | all 5 plugins |
| `Finalizable` | Cleanup after command | (available, unused) |

### Provider Resolution

`plugin.ResolveProvider()` — priority: CI env detection → `TERRACI_PROVIDER` env → single registered → error.

Core has zero knowledge of specific provider names. Plugins self-identify via `DetectEnv()` and `ProviderName()`.

### `xterraci build` — Custom Binary Builder

```bash
xterraci build                                          # all built-in plugins
xterraci build --with github.com/myco/terraci-plugin-X  # add external
xterraci build --without cost                           # exclude built-in
xterraci build --output ./bin/terraci                   # custom output
```

Generates `main.go` with selected plugin imports, runs `go mod init/get/tidy/build`.

## Configuration (.terraci.yaml)

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

exclude: ["*/test/*", "*/sandbox/*"]
include: []

library_modules:
  paths: ["_modules", "shared/modules"]

plugins:
  gitlab:
    image:
      name: hashicorp/terraform:1.6
    terraform_binary: terraform
    plan_enabled: true
    auto_approve: false
    job_defaults:
      tags: [terraform, docker]
    mr:
      comment: { enabled: true }
      summary_job:
        image: { name: "ghcr.io/edelwud/terraci:latest" }

  # github:
  #   terraform_binary: terraform
  #   runs_on: ubuntu-latest
  #   plan_enabled: true
  #   permissions: { contents: read, pull-requests: write }
  #   pr:
  #     comment: { enabled: true }

  # cost:
  #   enabled: true
  #   cache_dir: ~/.terraci/pricing
  #   cache_ttl: "24h"
  #   show_in_comment: true

  # policy:
  #   enabled: true
  #   sources:
  #     - path: terraform
  #   namespaces: [terraform]
  #   on_failure: block
```

Core config knows only: `structure`, `exclude`, `include`, `library_modules`, `plugins` (opaque YAML map). All provider/feature config lives inside `plugins:` — each plugin owns its config types.

## Core Data Model

**Module** (`discovery.Module`) — central type representing a Terraform module:
- Dynamic components: `components map[string]string` + `segments []string` — driven by config pattern
- `Get(name)` → component value by name (e.g., `m.Get("service")`, `m.Get("environment")`)
- `ID()` → `RelativePath` (filesystem path is the canonical ID)
- No hardcoded field names — any pattern like `{team}/{project}/{component}` works

**PatternSegments** (`config.PatternSegments`) — parsed from `structure.pattern`:
- `ParsePattern("{service}/{environment}/{region}/{module}")` → `["service", "environment", "region", "module"]`

## Data Flow

### Generate pipeline (main workflow)
1. `workflow.Run(ctx, opts)` → scan → filter → parse → build graph (with plugin hooks at each phase)
2. *(if `--changed-only`)* `ChangeDetectionProvider.DetectChangedModules()` → `GetAffectedModulesWithLibraries()`
3. `plugin.ResolveProvider()` → `GeneratorProvider.NewGenerator()` → produce CI YAML
4. `pipeline.BuildModuleEnvVars()` creates `TF_<SEGMENT>` env vars from module segments

### Summary (`terraci summary`)
1. `ci.ScanPlanResults()` → collect plan artifacts
2. `ExecutionContext` created with plan results
3. `SummaryContributor` plugins called in order (cost enriches costs, policy loads results)
4. `GeneratorProvider.NewCommentService()` → `UpsertComment(plans, policySummary)`

### Init wizard (`terraci init`)
1. Core form groups: Basics (provider, binary), Structure (pattern), Pipeline Options
2. `InitContributor` plugins contribute dynamic form groups (GitLab image, GitHub runner, cost toggle, etc.)
3. `BuildConfigFromPlugins(pattern, pluginConfigs)` assembles final config
4. Live YAML preview in TUI

### Static evaluation engine
- 30+ Terraform built-in functions evaluated at parse time
- Multi-pass locals resolution with `abspath(path.module)` support
- Variables from defaults + tfvars

### Cost estimation (cost plugin)
1. `terraform/plan.ParseJSON()` → parse plan.json
2. Estimator with handler registry dispatches by `CostCategory` (Standard/Fixed/UsageBased)
3. AWS Bulk Pricing API with TTL cache
4. Concurrent `EstimateModules()` via errgroup

### Policy checks (policy plugin)
1. OPA engine loads .rego files in single bundle
2. Multiple namespaces, per-module `**` glob overwrites
3. `Checker.ShouldBlock()` determines pipeline failure

## Key Patterns

- **No global state**: `App` struct holds config/workDir; commands are factory functions
- **Plugin-first**: core is provider-agnostic; all provider/feature logic in `plugins/`
- **Compile-time extensibility**: blank imports register plugins; `xterraci build` for custom binaries
- **Shared workflow**: `workflow.Run(ctx, opts)` with hook injection at 5 phases
- **Lazy AppContext**: `AppContext.Ensure()` refreshes from App state — safe for early command registration
- **ExecutionContext**: shared mutable state for inter-plugin communication during summary
- **InitContributor**: plugins contribute form fields + config building logic to init wizard
- **Plugin directory convention**: `plugin.go` (contract + re-exports) + `internal/` (implementation)
- **Context propagation**: `context.Context` flows through Scanner, Parser, DependencyExtractor
- **Typed errors**: `pkg/errors` with `Unwrap()` support
- **Generic filtering**: `--filter key=value` works with any segment name
- **Pattern-aware modules**: `structure.pattern` defines segment names; no hardcoded field names

## CLI Commands

```bash
terraci generate -o .gitlab-ci.yml                      # Generate pipeline
terraci generate --changed-only --base-ref main          # Only changed modules
terraci generate --plan-only                             # Plan jobs only
terraci generate --filter environment=prod               # Filter by segment

terraci validate                             # Validate config and structure
terraci graph --format dot -o deps.dot       # DOT graph
terraci graph --stats                        # Fan-in/fan-out stats

terraci init                                 # Interactive TUI wizard
terraci init --ci --provider github          # Non-interactive

terraci cost                                 # Estimate AWS costs (cost plugin)
terraci cost --module <path> --output json

terraci summary                              # Post MR/PR comment (CI only)

terraci policy pull                          # Download policies (policy plugin)
terraci policy check                         # Check plans against policies

terraci schema                               # Generate JSON schema
terraci version                              # Version + plugin info
```

**Global flags:** `-c/--config`, `-d/--dir`, `-v/--verbose`

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
