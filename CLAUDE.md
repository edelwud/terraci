# TerraCi

CLI tool for analyzing Terraform projects, building dependency graphs, generating CI pipelines (GitLab CI + GitHub Actions), and estimating AWS costs.

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
├── main.go                     # Entry point — calls cmd.NewRootCmd()
└── cmd/
    ├── app.go                  # App struct — shared state for all commands
    ├── root.go                 # NewRootCmd() factory, PersistentPreRunE, subcommand registration
    ├── generate.go             # Pipeline generation (main workflow)
    ├── filters.go              # filterFlags struct — shared filter flags and helpers
    ├── validate.go             # Config/project validation
    ├── graph.go                # Dependency graph visualization
    ├── init.go                 # Config initialization (entry point)
    ├── init_config.go          # initOptions — shared config builder for CLI/TUI
    ├── init_tui.go             # Interactive TUI wizard (bubbletea/huh)
    ├── cost.go                 # AWS cost estimation command
    ├── summary.go              # MR/PR comment posting (CI only)
    ├── policy.go               # OPA policy checks (pull, check)
    ├── schema.go               # JSON schema generation
    ├── completion.go           # Shell completion
    ├── man.go                  # Man page generation
    └── version.go              # Version info

internal/
├── workflow/
│   └── module_workflow.go      # Shared orchestration: scan → filter → parse → build graph
├── discovery/
│   ├── module.go               # Module struct (dynamic components + segments)
│   ├── scanner.go              # Scanner.Scan(ctx) — directory walk entry point
│   ├── collector.go            # moduleCollector — walk logic, context cancellation
│   ├── index.go                # ModuleIndex — fast lookups by ID/path/name
│   └── testing.go              # TestModule() helper for tests
├── parser/
│   ├── types.go                # Parser, ParsedModule, RemoteStateRef, ModuleCall
│   ├── hcl.go                  # ParseModule(ctx), multi-pass locals evaluation, extractors
│   ├── resolve.go              # ResolveWorkspacePath, for_each resolution
│   └── dependency.go           # DependencyExtractor, ExtractAllDependencies(ctx)
├── graph/
│   ├── dependency.go           # DependencyGraph, Node, edges, traversal, library usage
│   ├── algorithms.go           # TopologicalSort, ExecutionLevels, DetectCycles
│   ├── affected.go             # GetAffectedModules, library changes, combined
│   ├── visualize.go            # ToDOT (clustered), ToPlantUML (nested groups)
│   └── stats.go                # GetStats (fan-in/fan-out, modules per level)
├── filter/
│   └── glob.go                 # GlobFilter, SegmentFilter, CompositeFilter, Apply()
├── git/
│   ├── client.go               # Git client, ref resolution, fetch
│   ├── diff.go                 # GetChangedFiles, diffCommits, extractPaths
│   └── detector.go             # ChangedModulesDetector, isTerraformRelated
├── ci/
│   ├── types.go                # ModulePlan, PlanResult, CommentData, PolicySummary
│   ├── comment.go              # CommentRenderer — shared PR/MR comment markdown
│   ├── helpers.go              # HasReportableChanges — shared on_changes_only logic
│   ├── plan_result.go          # ScanPlanResults, ParseModulePathComponents
│   └── service.go              # CommentService interface
├── terraform/
│   ├── eval/
│   │   ├── context.go          # NewContext() — path.module as abspath, SafeObjectVal
│   │   └── functions.go        # 30+ Terraform functions (split, element, length, abspath, lookup, join, format, etc.)
│   └── plan/
│       ├── types.go            # ParsedPlan, ResourceChange, AttrDiff
│       ├── parser.go           # ParseJSON, countAction, buildAttrDiff
│       └── maputil.go          # Nested map utilities (toMap, getNestedValue, formatValue)
├── pipeline/
│   ├── pipeline.go             # Generator and GeneratedPipeline interfaces
│   ├── env.go                  # BuildModuleEnvVars — shared TF_* env var builder
│   ├── gitlab/
│   │   ├── generator.go        # GitLab CI Generator
│   │   └── types.go            # Pipeline, Job, ImageConfig, Secret, Rule
│   └── github/
│       ├── generator.go        # GitHub Actions Generator
│       └── types.go            # Workflow, Job, Step
├── github/
│   ├── client.go               # GitHub API client (go-github)
│   ├── context.go              # DetectPRContext from env vars
│   └── pr_service.go           # PRService — NewPRService(cfg, client, ctx) with DI
├── gitlab/
│   ├── client.go               # GitLab API Client, MRContext
│   ├── mr_service.go           # MRService — NewMRService(cfg, client, ctx) with DI
│   └── comment.go              # FindTerraCIComment
├── policy/
│   ├── engine.go               # OPA Engine — loads all .rego in single bundle
│   ├── checker.go              # Checker — CheckModule(), CheckAll(), overwrite reclassification
│   ├── result.go               # Result, Violation, Summary
│   ├── source.go               # Source interface, Puller
│   ├── source_path.go          # PathSource
│   ├── source_git.go           # GitSource
│   └── source_oci.go           # OCISource
└── cost/
    ├── types.go                # ResourceCost (with CostErrorKind), ModuleCost, EstimateResult
    ├── estimator.go            # Estimator — EstimateModule(), EstimateModules() (concurrent)
    ├── factory.go              # NewEstimatorFromConfig(cfg) — shared estimator creation
    ├── registry.go             # init() — registers all handlers from subpackages (breaks import cycle)
    ├── aws/
    │   ├── registry.go         # Registry, ResourceHandler interface, CostCategory
    │   ├── attrs.go            # GetStringAttr, GetFloatAttr, GetIntAttr, GetBoolAttr
    │   ├── calc.go             # HourlyCost, ScaledHourlyCost, FixedMonthlyCost
    │   ├── region.go           # HoursPerMonth, ResolveRegionName, ResolveUsagePrefix
    │   ├── lookup.go           # LookupBuilder — constructs PriceLookup with auto-location
    │   ├── constants.go        # Shared VolumeType* constants
    │   ├── ec2/                # InstanceHandler, EBSHandler, EIPHandler, NATHandler
    │   ├── rds/                # InstanceHandler, ClusterHandler, ClusterInstanceHandler
    │   ├── elb/                # ALBHandler, ClassicHandler
    │   ├── elasticache/        # ClusterHandler, ReplicationGroupHandler
    │   ├── eks/                # ClusterHandler, NodeGroupHandler
    │   ├── serverless/         # LambdaHandler, DynamoDBHandler, SQSHandler, SNSHandler
    │   └── storage/            # S3Handler, LogGroupHandler, AlarmHandler, SecretsManagerHandler, KMSHandler, Route53Handler
    └── pricing/
        ├── types.go            # ServiceCode, PriceIndex, Price, PriceLookup, RegionMapping
        ├── fetcher.go          # AWS Bulk Pricing API fetcher + unexported AWS API response types
        └── cache.go            # TTL-based local pricing cache with mutex + integrity check

pkg/
├── config/
│   ├── config.go               # Config, Load(), Validate(), matchGlob with ** support
│   ├── pattern.go              # ParsePattern, PatternSegments
│   └── schema.go               # JSON schema generation
├── errors/
│   └── errors.go               # Typed errors: ConfigError, ScanError, ParseError, NoModulesError, etc.
└── log/log.go                  # Structured logging (wraps caarlos0/log)
```

## Architecture

### Command layer (`cmd/terraci/cmd/`)

Commands use dependency injection via the `App` struct — no global mutable state:

- `NewRootCmd(version, commit, date)` creates `App` and returns the root `*cobra.Command`
- Each command is a factory: `func newXxxCmd(app *App) *cobra.Command`
- Command-specific flags are local variables within the factory function scope
- Shared filter flags use `filterFlags` struct with methods: `register()`, `applyFilters()`, `workflowOptions()`
- `PersistentPreRunE` loads config into `app.Config` and sets `app.WorkDir`

### Module workflow (`internal/workflow/`)

`workflow.Run(ctx, opts)` encapsulates the common pipeline used by generate, graph, and validate:

1. `Scanner.Scan(ctx)` → discover modules
2. `filter.Apply()` → exclude/include/segment filters
3. `DependencyExtractor.ExtractAllDependencies(ctx)` → parse HCL, resolve remote_state
4. `graph.BuildFromDependencies()` → build DAG

Returns `Result` with AllModules, FilteredModules, FullIndex, FilteredIndex, Graph, Dependencies, Warnings.

### Context propagation

`context.Context` flows through all I/O-bound operations:
- `Scanner.Scan(ctx)` — checks `ctx.Err()` during directory walk
- `Parser.ParseModule(ctx, path)` — checks context before parsing
- `DependencyExtractor.ExtractDependencies(ctx, module)` / `ExtractAllDependencies(ctx)` — propagates to concurrent goroutines via errgroup

### Typed errors (`pkg/errors/`)

- `ConfigError{Path, Err}` — config loading/validation
- `ScanError{Dir, Err}` — module discovery failures
- `ParseError{Module, Err}` — HCL parsing errors
- `NoModulesError{Dir}` — no modules found
- `PolicyError{Module, Violations}` — policy check failures
- `CostError{Module, Err}` — cost estimation errors
- `GraphError{Cycles}` — dependency graph issues

All support `errors.Is()`/`errors.As()` via `Unwrap()`.

## Core Data Model

**Module** (`discovery.Module`) — central type representing a Terraform module:
- Dynamic components: `components map[string]string` + `segments []string` — driven by config pattern
- `Get(name)` → component value by name (e.g., `m.Get("service")`, `m.Get("environment")`)
- `LeafValue()` → value of last pattern segment (the "module" equivalent)
- `ID()` → `RelativePath` (filesystem path is the canonical ID)
- `ContextPrefix()` → all segments except last, joined (for context-relative lookups)
- `Name()` → leaf value + `/submodule` if present
- Discovered by `Scanner.Scan(ctx)` using configurable pattern segments
- No hardcoded field names — any pattern like `{team}/{project}/{component}` works

**PatternSegments** (`config.PatternSegments`) — parsed from `structure.pattern`:
- `ParsePattern("{service}/{environment}/{region}/{module}")` → `["service", "environment", "region", "module"]`
- Stored in `config.Structure.Segments` (parsed at config load time)
- Passed to Scanner, Parser, and pipeline generators

## Data Flow

### Generate pipeline (main workflow)
1. `workflow.Run(ctx, opts)` → scan, filter, parse, build graph (shared by generate/graph/validate)
2. *(if `--changed-only`)* Git diff → detect changed modules → `GetAffectedModulesWithLibraries()`
3. `pipeline.Generator.Generate()` → produce GitLab CI or GitHub Actions YAML
4. `pipeline.BuildModuleEnvVars()` creates `TF_<SEGMENT>` env vars from module segments
5. *(in CI)* `ci.CommentService.UpsertComment()` → post plan/policy/cost summary to MR/PR

### Static evaluation engine
- 30+ Terraform built-in functions: `split`, `element`, `length`, `abspath`, `lookup`, `join`, `format`, `lower`, `upper`, `trimprefix`, `trimsuffix`, `replace`, `concat`, `contains`, `keys`, `values`, `merge`, `flatten`, `distinct`, `tostring`, `tonumber`, `tobool`, `max`, `min`, `ceil`, `floor`
- `path.module` returns absolute path — enables `abspath(path.module)` pattern
- Multi-pass locals evaluation: locals referencing other locals, path.module, and functions are resolved iteratively (up to 10 passes)
- Variables loaded from: `default` values (any type), `terraform.tfvars`, `*.auto.tfvars`
- Custom `lookupFunc` handles both map and object types (stdlib version doesn't)

### Provider auto-detection
`config.ResolveProvider(cfg)`: explicit `provider` field → `GITHUB_ACTIONS` env → `GITLAB_CI` env → default `"gitlab"`

### Cost estimation
1. `terraform/plan.ParseJSON()` → parse plan.json into ResourceChange list
2. `cost.NewEstimatorFromConfig(cfg)` → create estimator with cache settings
3. `cost.Estimator.ValidateAndPrefetch()` → identify required services (skips Fixed/UsageBased), fetch pricing
4. Per resource: dispatch by `handler.Category()`:
   - `CostCategoryStandard` → `BuildLookup()` → `pricing.Cache` → `CalculateCost()` (full API path)
   - `CostCategoryFixed` → `CalculateCost(nil, attrs)` directly (no API call)
   - `CostCategoryUsageBased` → return $0 with `CostErrorUsageBased` (SQS, SNS, S3, CloudWatch Logs)
5. For update/replace actions: `getBeforeAttrs()` and `getAfterAttrs()` compute separate before/after costs
6. `EstimateModules()` runs concurrently via errgroup (limit 4)
7. `terraci cost` command runs estimation locally; `terraci summary` includes costs in MR/PR comments
8. Error classification: `CostErrorKind` (NoHandler, UsageBased, LookupFailed, APIFailure, NoPrice)

### Policy checks
1. `policy.Puller` downloads policies from sources (path/git/OCI)
2. `policy.Engine.Evaluate()` runs OPA against plan.json — all .rego files loaded in single bundle
3. Multiple namespaces supported (e.g., `terraform` + `compliance`)
4. `policy.Checker.CheckAll()` aggregates results → `Summary`
5. Per-module overwrites via `**` glob patterns: `on_failure: warn` reclassifies failures as warnings, `enabled: false` skips evaluation
6. `Checker.ShouldBlock()` determines if pipeline should fail

## CLI Commands

```bash
terraci generate -o .gitlab-ci.yml                      # Generate GitLab pipeline
terraci generate -o .github/workflows/terraform.yml     # Generate GitHub Actions
terraci generate --changed-only --base-ref main          # Only changed modules
terraci generate --plan-only                             # Plan jobs only
terraci generate --filter environment=prod               # Filter by any segment
terraci generate --exclude "*/test/*"                    # Glob exclusion

terraci validate                             # Validate config and structure
terraci graph --format dot -o deps.dot       # DOT graph (clustered subgraphs)
terraci graph --format plantuml              # PlantUML (nested groups)
terraci graph --format levels                # Execution levels with dep hints
terraci graph --stats                        # Fan-in/fan-out, modules per level
terraci graph --module <id> --dependents     # Show dependents

terraci init                                 # Interactive TUI wizard
terraci init --ci                            # Non-interactive with defaults
terraci init --provider github               # GitHub Actions preset

terraci cost                                 # Estimate AWS costs from plan.json
terraci cost --module <path>                 # Single module cost
terraci cost --output json                   # JSON output

terraci summary                              # Post MR/PR comment (CI only)

terraci policy pull                          # Download policies
terraci policy check                         # Check all modules

terraci schema                               # Generate JSON schema
terraci completion bash|zsh|fish             # Shell completions
terraci version                              # Version + OPA version
```

**Global flags:** `-c/--config`, `-d/--dir`, `-v/--verbose`

**Shared filter flags** (generate, graph, validate):
- `-x/--exclude` — glob patterns
- `-i/--include` — glob patterns
- `-f/--filter key=value` — filter by any segment name

## Configuration (.terraci.yaml)

```yaml
provider: gitlab                        # or "github" (auto-detected from CI env)

structure:
  pattern: "{service}/{environment}/{region}/{module}"

exclude: ["*/test/*", "*/sandbox/*"]
include: []

library_modules:
  paths: ["_modules", "shared/modules"]

# GitLab CI (omitted when provider=github)
gitlab:
  image:
    name: hashicorp/terraform:1.6
  terraform_binary: "terraform"         # or "tofu"
  plan_enabled: true
  auto_approve: false
  job_defaults:
    tags: [terraform, docker]
  mr:
    comment: { enabled: true }
    summary_job:
      image: { name: "ghcr.io/edelwud/terraci:latest" }

# GitHub Actions (omitted when provider=gitlab)
github:
  terraform_binary: "terraform"
  runs_on: "ubuntu-latest"
  plan_enabled: true
  auto_approve: false
  permissions: { contents: read, pull-requests: write }
  pr:
    comment: { enabled: true }

policy:
  enabled: false
  sources:
    - path: terraform                   # directory name = Rego package name
    - path: compliance
  namespaces: [terraform, compliance]
  on_failure: block                     # block, warn, ignore
  overwrites:
    - match: "**/sandbox/**"
      on_failure: warn                  # reclassify failures as warnings
    - match: "legacy/**"
      enabled: false                    # skip policy checks

cost:
  enabled: false
  show_in_comment: true
  cache_dir: ~/.terraci/pricing
  cache_ttl: "24h"
```

## Key Patterns

- **No global state**: `App` struct holds config/workDir; commands are factory functions `newXxxCmd(app *App)` returning `*cobra.Command`; filter flags encapsulated in `filterFlags` struct
- **Shared workflow**: `workflow.Run(ctx, opts)` eliminates repeated scan→filter→parse→graph code across generate/graph/validate commands
- **Context propagation**: `context.Context` flows through Scanner, Parser, DependencyExtractor for cancellation/timeout support
- **Dependency injection**: `NewMRService(cfg, client, ctx)` / `NewPRService(cfg, client, ctx)` accept dependencies; `FromEnv` convenience constructors for production
- **Typed errors**: `pkg/errors` provides `ConfigError`, `ScanError`, `ParseError`, `NoModulesError` etc. with `Unwrap()` support
- **Pattern-aware modules**: `structure.pattern` defines segment names and order; Module uses `components map[string]string` — no hardcoded field names
- **Static evaluation**: 30+ Terraform functions evaluated at parse time; multi-pass locals resolution with `abspath(path.module)` support; variables from defaults + tfvars
- **Shared env vars**: `pipeline.BuildModuleEnvVars()` creates `TF_<SEGMENT>` from module segments — shared by GitLab and GitHub generators
- **Multi-provider**: `pipeline.Generator` interface with GitLab and GitHub implementations; provider auto-detected from CI env
- **Decoupled CI layer**: `internal/ci/` has provider-agnostic types (`PolicySummary`, `CommentData`); no dependency on `internal/policy/`; `internal/gitlab/` and `internal/github/` are thin provider wrappers
- **Generic filtering**: `SegmentFilter{Segment, Values}` replaces hardcoded filters; `--filter key=value` CLI flag works with any segment name
- **Dependencies**: resolved from `terraform_remote_state` data blocks; `for_each` with ternary + for-expressions + lookup on objects; `matchPathToModule` uses strategy chain
- **Graph visualization**: DOT with clustered subgraphs and short labels; PlantUML with nested region grouping; stats with fan-in/fan-out top-5 and modules per level
- **Policy checks**: OPA v1 Rego; multiple namespaces; per-module `**` glob overwrites (warn/disable); single-bundle loading; `deny` → failures, `warn` → warnings
- **Cost estimation**: Handlers grouped by service in subpackages (`aws/ec2/`, `aws/rds/`, etc.); `CostCategory` classifies handlers (Standard/Fixed/UsageBased); `LookupBuilder` + `HourlyCost()`/`ScaledHourlyCost()` eliminate handler boilerplate; `usagetype` filtering prevents ambiguous pricing matches (e.g., ElastiCache ExtendedSupport variants); concurrent `EstimateModules()` via errgroup; `CostErrorKind` classifies estimation failures; `cost/registry.go` init() breaks import cycle between `aws/` and subpackages
- **MR/PR comments**: upserted via `<!-- terraci-plan-comment -->` marker; `ci.HasReportableChanges()` shared on_changes_only logic
- **Config**: `matchGlob` with `**` multi-segment pattern support; `image:` as object with `name:` field; no `backend:` section (removed — dead code)
- **Interactive init**: bubbletea TUI with live YAML preview; `initOptions` shared between CLI and TUI modes
- **Testing**: all AWS pricing calls mocked via httptest; `Fetcher.Client`/`BaseURL` exported for injection

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
