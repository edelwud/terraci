# TerraCi

CLI tool for analyzing Terraform projects, building dependency graphs, generating GitLab CI pipelines, and estimating AWS costs.

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
├── main.go                     # Entry point
└── cmd/                        # Cobra commands
    ├── root.go                 # Root command, global flags, config loading
    ├── generate.go             # Pipeline generation (main workflow)
    ├── validate.go             # Config/project validation
    ├── graph.go                # Dependency graph visualization
    ├── init.go                 # Config initialization
    ├── summary.go              # MR comment posting (CI only)
    ├── policy.go               # OPA policy checks (pull, check)
    ├── schema.go               # JSON schema generation
    ├── completion.go           # Shell completion
    ├── man.go                  # Man page generation
    └── version.go              # Version info

internal/
├── discovery/module.go         # Module, Scanner, ModuleIndex
├── parser/
│   ├── hcl.go                  # Parser, ParsedModule, RemoteStateRef, ModuleCall
│   └── dependency.go           # DependencyExtractor, Dependency, LibraryDependency
├── graph/dependency.go         # DependencyGraph, Node, BuildFromDependencies
├── filter/glob.go              # GlobFilter for pattern-based filtering
├── git/diff.go                 # Git Client, ChangedModulesDetector
├── terraform/
│   ├── eval/                   # HCL evaluation context with Terraform functions
│   │   ├── context.go          # NewContext() — locals, variables, path
│   │   └── functions.go        # Terraform function implementations (lookup)
│   └── plan/
│       └── parser.go           # ParseJSON() → ParsedPlan, ResourceChange, AttrDiff
├── pipeline/
│   ├── pipeline.go             # Generator and GeneratedPipeline interfaces
│   └── gitlab/
│       ├── generator.go        # GitLab CI Generator implementation
│       └── types.go            # Pipeline, Job, ImageConfig, Secret, Rule, etc.
├── gitlab/
│   ├── client.go               # GitLab API Client, MRContext
│   ├── mr_service.go           # MRService — upserts MR comments
│   ├── comment.go              # CommentRenderer, ModulePlan, CommentData
│   └── plan_result.go          # ScanPlanResults from plan.txt artifacts
├── policy/
│   ├── engine.go               # OPA Engine, OPAVersion()
│   ├── checker.go              # Checker — CheckModule(), CheckAll(), ShouldBlock()
│   ├── result.go               # Result, Violation, Summary
│   ├── source.go               # Source interface, Puller
│   ├── source_path.go          # PathSource
│   ├── source_git.go           # GitSource
│   └── source_oci.go           # OCISource
└── cost/
    ├── types.go                # ResourceCost, ModuleCost, EstimateResult, FormatCost()
    ├── estimator.go            # Estimator — EstimateModule(), EstimateModules()
    ├── aws/
    │   ├── registry.go         # Registry, ResourceHandler interface
    │   ├── ec2.go              # EC2, EBS, EIP, NAT Gateway handlers
    │   ├── rds.go              # RDS instance/cluster handlers
    │   ├── elb.go              # ALB, Classic LB handlers
    │   ├── elasticache.go      # ElastiCache cluster/replication handlers
    │   ├── eks.go              # EKS cluster/node group handlers
    │   ├── serverless.go       # Lambda, DynamoDB, SQS, SNS, Secrets Manager
    │   └── storage.go          # S3, EBS optimization, VPC endpoints
    └── pricing/
        ├── types.go            # ServiceCode, PriceIndex, Price, PriceLookup
        ├── fetcher.go          # AWS Bulk Pricing API fetcher
        └── cache.go            # TTL-based local pricing cache

pkg/
├── config/
│   ├── config.go               # Config, Load(), Validate(), all config types
│   └── schema.go               # JSON schema generation
└── log/log.go                  # Structured logging (wraps caarlos0/log)
```

## Core Data Model

**Module** (`discovery.Module`) — central type representing a Terraform module:
- Fields: Service, Environment, Region, Module, Submodule, Path, RelativePath, Parent, Children
- `ID()` → `"service/env/region/module[/submodule]"`
- Discovered by `Scanner.Scan()` from directory pattern `service/environment/region/module[/submodule]`
- Depth 4 = base module, depth 5 = submodule

## Data Flow

### Generate pipeline (main workflow)
1. `Scanner.Scan()` → discover modules by directory structure
2. `filter.GlobFilter` → apply include/exclude patterns
3. `DependencyExtractor.ExtractAllDependencies()` → parse HCL, resolve remote_state refs
4. `graph.BuildFromDependencies()` → build DAG, detect cycles, compute execution levels
5. `DependencyGraph.AddLibraryUsage()` → track non-executable shared module usage
6. *(if `--changed-only`)* Git diff → detect changed modules → `GetAffectedModulesWithLibraries()`
7. `gitlab.Generator.Generate()` → produce GitLab CI YAML
8. *(in CI)* `MRService.UpsertComment()` → post plan/policy/cost summary to MR

### Cost estimation
1. `terraform/plan.ParseJSON()` → parse plan.json into ResourceChange list
2. `cost.Estimator.ValidateAndPrefetch()` → identify required AWS services, fetch pricing
3. Per resource: `aws.Registry` → find `ResourceHandler` → `BuildLookup()` → `pricing.Cache` → `CalculateCost()`
4. Aggregate into `ModuleCost` with before/after/diff

### Policy checks
1. `policy.Puller` downloads policies from sources (path/git/OCI)
2. `policy.Engine.Evaluate()` runs OPA against plan.json (v1 Rego syntax)
3. `policy.Checker.CheckAll()` aggregates results → `Summary`
4. `Checker.ShouldBlock()` determines if pipeline should fail

## CLI Commands

```bash
terraci generate -o .gitlab-ci.yml          # Generate pipeline
terraci generate --changed-only --base-ref main  # Only changed modules
terraci generate --plan-only                 # Plan jobs only, no apply
terraci generate --exclude "*/test/*" --environment prod

terraci validate                             # Validate config and structure
terraci graph --format dot -o deps.dot       # DOT graph output
terraci graph --format levels                # Execution levels
terraci graph --module <id> --dependents     # Show dependents

terraci init                                 # Create .terraci.yaml
terraci summary                              # Post MR comment (CI only)

terraci policy pull                          # Download policies
terraci policy check                         # Check all modules
terraci policy check --module <id> --output json

terraci schema                               # Generate JSON schema
terraci completion bash|zsh|fish             # Shell completions
terraci version                              # Version + OPA version
```

**Global flags:** `-c/--config` (config path), `-d/--dir` (working dir), `-v/--verbose`

## Configuration (.terraci.yaml)

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4                          # auto-calculated from pattern
  max_depth: 5                          # min_depth+1 if allow_submodules
  allow_submodules: true

exclude: ["*/test/*", "*/sandbox/*"]
include: []                             # if set, only matching modules

library_modules:
  paths: ["_modules", "shared/modules"] # non-executable shared modules

gitlab:
  image: "hashicorp/terraform:1.6"
  terraform_binary: "terraform"         # or "tofu"
  stages_prefix: "deploy"              # produces deploy-plan-0, deploy-apply-0
  parallelism: 5
  plan_enabled: true
  plan_only: false
  auto_approve: false
  cache_enabled: true
  init_enabled: true
  variables: {}
  rules: []                             # workflow-level rules

  job_defaults:                         # applied to all jobs
    image: ...
    tags: [terraform, docker]
    id_tokens: {}                       # OIDC tokens
    secrets: {}                         # Vault secrets
    before_script: []
    after_script: []
    artifacts: { paths: [], expire_in: "1 day" }
    rules: []
    variables: {}

  overwrites:                           # per job-type overrides
    - type: plan|apply
      image: ...                        # same fields as job_defaults

  mr:
    comment:
      enabled: true
      on_changes_only: false
      include_details: true
    labels: ["{service}", "{environment}"]
    summary_job:
      image: { name: "ghcr.io/edelwud/terraci:latest" }
      tags: []

backend:
  type: s3                              # s3, gcs, azurerm, local, remote
  bucket: "..."
  region: "..."
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"

policy:
  enabled: false
  sources:
    - path: policies
    - git: https://... ref: main
    - oci: oci://...
  namespaces: [terraform]
  on_failure: block                     # block, warn, ignore
  on_warning: warn
  show_in_comment: true
  cache_dir: .terraci/policies
  overwrites:
    - match: "*/sandbox/*"
      on_failure: warn

cost:
  enabled: false
  cache_dir: ~/.terraci/pricing
  cache_ttl: "24h"
  show_in_comment: true
```

Config files searched: `.terraci.yaml`, `.terraci.yml`, `terraci.yaml`, `terraci.yml`

## Key Patterns

- **Module discovery**: directory depth determines modules (min=4 base, max=5 submodules)
- **Dependencies**: resolved from `terraform_remote_state` data blocks; `for_each` expands to multiple deps
- **Graph algorithms**: Kahn's topological sort, DFS cycle detection, execution level grouping
- **Pipeline generation**: `pipeline.Generator` interface with GitLab implementation
- **Cost estimation**: plugin registry pattern — `aws.ResourceHandler` interface per resource type
- **MR comments**: upserted via `<!-- terraci-plan-comment -->` marker
- **Policy checks**: OPA v1 Rego syntax (`deny contains msg if {...}`), results saved to `.terraci/policy-results.json`
- **Config precedence**: `JobDefaults` → `JobOverwrite` (plan/apply specific) → generated job
- **Image config**: supports both string `"image:tag"` and object `{name: ..., entrypoint: [...]}`
- **Vault secrets**: supports string shorthand `"path/field@namespace"` and full object syntax

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/hashicorp/hcl/v2` | HCL parsing |
| `github.com/zclconf/go-cty` | CTY types for HCL |
| `github.com/hashicorp/terraform-json` | Terraform plan JSON types |
| `go.yaml.in/yaml/v4` | YAML serialization |
| `gitlab.com/gitlab-org/api/client-go` | GitLab API client |
| `github.com/open-policy-agent/opa` | Embedded OPA engine |
| `github.com/go-git/go-git/v6` | Git operations |
| `oras.land/oras-go/v2` | OCI registry operations |
| `github.com/invopop/jsonschema` | JSON schema generation |
| `github.com/caarlos0/log` | Structured logging |
| `golang.org/x/sync` | Concurrency utilities |
