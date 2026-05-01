<p align="center">
  <img src="docs/public/logo.svg" alt="TerraCi" width="120" height="120">
</p>

<h1 align="center">TerraCi</h1>

<p align="center">
  <strong>Terraform Pipeline Generator for GitLab CI & GitHub Actions</strong><br>
  Analyze dependencies, estimate costs, enforce policies, and generate optimal pipelines for Terraform/OpenTofu monorepos
</p>

<p align="center">
  <a href="https://github.com/edelwud/terraci/releases"><img src="https://img.shields.io/github/v/release/edelwud/terraci?style=flat-square&color=5f67ee" alt="Release"></a>
  <a href="https://github.com/edelwud/terraci/actions"><img src="https://github.com/edelwud/terraci/actions/workflows/build.yml/badge.svg" alt="Build"></a>
  <a href="https://github.com/edelwud/terraci/blob/main/LICENSE"><img src="https://img.shields.io/github/license/edelwud/terraci?style=flat-square" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/edelwud/terraci"><img src="https://goreportcard.com/badge/github.com/edelwud/terraci?style=flat-square" alt="Go Report"></a>
</p>

<p align="center">
  <a href="https://edelwud.github.io/terraci">Documentation</a> &bull;
  <a href="#installation">Installation</a> &bull;
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="https://github.com/edelwud/terraci/tree/main/examples">Examples</a>
</p>

---

## Why TerraCi?

Managing Terraform in a monorepo is painful:

- **Dependencies** between modules must be respected — EKS can't deploy before VPC
- **Manual pipelines** are tedious to write and a nightmare to maintain
- **Full deployments** waste CI minutes when only one module changed
- **No visibility** into what a plan will cost or whether it violates policies

TerraCi solves all of this. Point it at your repo, and it generates correct, dependency-aware CI pipelines — with cost estimates and policy checks baked in.

## Features

<table>
<tr><td width="50%">

**Pipeline Generation**
- GitLab CI & GitHub Actions support
- Dependency-aware topological ordering
- Parallel execution of independent modules
- Plan + apply stages with manual approval gates
- Changed-only mode via git diff

</td><td>

**Intelligence**
- AWS cost estimation on every plan
- OPA policy enforcement (local, git, OCI sources)
- MR/PR comments with plan summaries, costs & policy results
- Dependency graph visualization (DOT/PlantUML)
- Terraform dependency resolution with lock file synchronization

</td></tr>
<tr><td>

**Flexibility**
- Terraform & OpenTofu support
- Configurable directory patterns (any segment names)
- Per-job overrides (image, secrets, tags, rules)
- OIDC tokens & Vault secrets integration
- Auto-detect CI provider from environment

</td><td>

**Developer Experience**
- Interactive TUI wizard for initialization
- Single binary, zero dependencies
- JSON schema for IDE autocomplete
- Shell completions (bash, zsh, fish)
- Dry-run mode to preview changes
- Custom plugins via `xterraci build --with`

</td></tr>
</table>

## Installation

```bash
# Homebrew
brew install edelwud/tap/terraci

# Go
go install github.com/edelwud/terraci/cmd/terraci@latest

# Docker
docker run --rm -v $(pwd):/workspace ghcr.io/edelwud/terraci:latest generate

# Binary — download from GitHub Releases
# https://github.com/edelwud/terraci/releases
```

## Quick Start

```bash
# Interactive setup wizard
terraci init

# Non-interactive with GitHub Actions
terraci init --ci --provider github

# Validate structure & dependencies
terraci validate

# Generate pipeline
terraci generate -o .gitlab-ci.yml                      # GitLab
terraci generate -o .github/workflows/terraform.yml     # GitHub Actions

# Only changed modules (CI)
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

## How It Works

```
 Your Repo                    TerraCi                          CI Pipeline
┌─────────────┐    ┌──────────────────────────┐    ┌──────────────────────────┐
│ platform/   │    │ 1. Discover modules      │    │                          │
│  prod/      │───>│ 2. Parse remote_state    │───>│  plan-vpc → apply-vpc    │
│   eu-west/  │    │ 3. Build dependency DAG  │    │       ↓           ↓      │
│    vpc/     │    │ 4. Topological sort      │    │  plan-eks   plan-rds     │
│    eks/     │    │ 5. Detect CI provider    │    │       ↓           ↓      │
│    rds/     │    │ 6. Generate YAML         │    │  apply-eks  apply-rds    │
└─────────────┘    └──────────────────────────┘    │                          │
                                                   │  Output: .gitlab-ci.yml  │
                                                   │     or   workflow.yml    │
                                                   └──────────────────────────┘
```

### 1. Module Discovery

Scans directories matching a configurable pattern (default: `{service}/{environment}/{region}/{module}`):

```
platform/prod/eu-central-1/vpc/           → Module: platform/prod/eu-central-1/vpc
platform/prod/eu-central-1/eks/           → Module: platform/prod/eu-central-1/eks
platform/prod/eu-central-1/ec2/rabbitmq/  → Submodule (depth 5)
```

The pattern is fully configurable — `{team}/{project}/{component}` works too.

### 2. Dependency Resolution

Dependencies are extracted from `terraform_remote_state` data sources. The `key` in the remote state config must mirror your directory structure (matching `structure.pattern`) — this is how TerraCi maps state file paths back to modules:

```hcl
# eks/main.tf — key follows the same {service}/{env}/{region}/{module} pattern
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "my-state"
    key    = "platform/prod/eu-central-1/vpc/terraform.tfstate"
    #        ^^^^^^^^^^^^^^^^^^^^^^^^^^^
    #        matches directory structure → TerraCi resolves: eks depends on vpc
    region = "eu-central-1"
  }
}
```

TerraCi evaluates Terraform functions statically (`split`, `element`, `length`, `abspath`, `lookup`, `join`, `format`, etc.) and resolves locals that depend on `path.module`. A common pattern that works out of the box:

```hcl
locals {
  path_arr    = split("/", abspath(path.module))
  service     = local.path_arr[length(local.path_arr) - 4]
  environment = local.path_arr[length(local.path_arr) - 3]
  region      = local.path_arr[length(local.path_arr) - 2]
}

data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = "${local.service}/${local.environment}/${local.region}/vpc/terraform.tfstate"
  }
}
```

> **Limitations:**
> - **Static analysis only** — TerraCi does not connect to remote backends or execute `terraform init`. Dependencies that rely on runtime values (e.g., `data.terraform_remote_state.X.outputs.Y` used as a key in another remote state) cannot be resolved. Derive your state keys from the filesystem path (`abspath(path.module)`) or explicit locals, not from other modules' outputs.
> - **Backend-aware matching** — When the `key` path alone is ambiguous (e.g., two modules with the same key in different buckets), TerraCi parses each module's `terraform { backend "s3" { ... } }` block and uses the `bucket` to disambiguate. This requires backend configuration to be defined in the module's `.tf` files (not solely via `-backend-config` CLI flags).

### 3. Pipeline Generation

Modules are topologically sorted and grouped into parallel execution levels:

```
Level 0: vpc                    (no dependencies)
Level 1: eks, rds               (depend on vpc — run in parallel)
Level 2: app                    (depends on eks)
```

Each level gets plan + apply stages. Apply requires the previous level to complete.

## Configuration

```yaml
# .terraci.yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

exclude:
  - "*/test/*"
  - "*/sandbox/*"

execution:
  binary: "terraform"           # or "tofu"
  plan_enabled: true

extensions:
  # GitLab CI (omit for GitHub Actions)
  gitlab:
    image: { name: "hashicorp/terraform:1.6" }
    auto_approve: false
    mr:
      comment: { enabled: true }

  # GitHub Actions (omit for GitLab CI)
  # github:
  #   runs_on: "ubuntu-latest"
  #   pr:
  #     comment: { enabled: true }

  # AWS cost estimation
  # cost:
  #   providers:
  #     aws: { enabled: true }

  # OPA policy checks
  # policy:
  #   enabled: true
  #   sources:
  #     - path: policies
  #   on_failure: block                  # block, warn

  # Dependency update checks
  # tfupdate:
  #   enabled: true
  #   target: all                        # all, modules, providers
  #   policy:
  #     bump: minor                      # patch, minor, major
```

> **Tip:** Add `# yaml-language-server: $schema=https://raw.githubusercontent.com/edelwud/terraci/main/terraci.schema.json` at the top of your `.terraci.yaml` for IDE autocomplete. Or run `terraci schema` to generate the schema locally.

## Commands

| Command | Description |
|---------|-------------|
| `terraci init` | Interactive TUI wizard to create `.terraci.yaml` |
| `terraci validate` | Validate project structure and dependencies |
| `terraci generate` | Generate CI pipeline (GitLab CI or GitHub Actions) |
| `terraci graph` | Visualize dependency graph (DOT, PlantUML, levels) |
| `terraci cost` | Estimate AWS costs from Terraform plan files |
| `terraci summary` | Post plan/cost/policy summary to MR/PR (CI) |
| `terraci policy pull` | Download policies from configured sources |
| `terraci policy check` | Evaluate plans against OPA policies |
| `terraci schema` | Generate JSON schema for config validation |
| `terraci tfupdate` | Resolve Terraform dependency versions and sync lock files |
| `terraci version` | Show version and embedded OPA version |

<details>
<summary><strong>CI integration patterns</strong></summary>

**GitLab CI** supports [generative pipelines](https://docs.gitlab.com/ci/pipelines/downstream_pipelines/) — TerraCi runs as a job inside a parent pipeline and generates a child pipeline that GitLab picks up automatically:

```yaml
# .gitlab-ci.yml (parent pipeline)
generate:
  stage: generate
  image: ghcr.io/edelwud/terraci:latest
  script:
    - terraci generate --changed-only --base-ref $CI_MERGE_REQUEST_DIFF_BASE_SHA -o generated.yml
  artifacts:
    paths: [generated.yml]

deploy:
  stage: deploy
  trigger:
    include:
      - artifact: generated.yml
        job: generate
```

**GitHub Actions** does not support dynamic workflow generation at runtime. Use a **pre-commit hook** to regenerate the workflow file and commit it alongside your changes:

```bash
# .husky/pre-commit or .git/hooks/pre-commit
terraci generate --changed-only --base-ref main -o .github/workflows/terraform.yml
git add .github/workflows/terraform.yml
```

</details>

<details>
<summary><strong>Common usage examples</strong></summary>

```bash
# Changed modules only (for MR/PR pipelines)
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml

# Filter by any segment name
terraci generate --filter environment=prod --filter service=platform

# Plan-only mode (no apply jobs)
terraci generate --plan-only

# Exclude patterns
terraci generate --exclude "*/sandbox/*" --exclude "*/test/*"

# Dependency graph as DOT
terraci graph --format dot -o deps.dot

# Show execution levels
terraci graph --format levels

# Show what depends on a module
terraci graph --module platform/prod/eu-central-1/vpc --dependents

# Policy check a specific module
terraci policy check --module platform/prod/eu-central-1/vpc --output json

# Dry run
terraci generate --dry-run

# Estimate AWS costs from plan.json files
terraci cost

# Cost for a single module
terraci cost --module platform/prod/eu-central-1/rds

# Cost as JSON
terraci cost --output json
```

JSON output now uses explicit resource `status` values:
- `exact` — fully priced at plan time
- `usage_estimated` — partly estimated from configured capacity
- `usage_unknown` — still unknown at plan time and needs runtime telemetry
- `unsupported` / `failed` — not priced, with optional `failure_kind` and `status_detail`

</details>

## Custom Plugins

Build a custom TerraCi binary with additional (or fewer) plugins using `xterraci`:

```bash
# Install xterraci
go install github.com/edelwud/terraci/cmd/xterraci@latest

# Add an external plugin
xterraci build --with github.com/myorg/terraci-plugin-slack

# Use a local plugin during development
xterraci build --with github.com/myorg/plugin=../my-plugin

# Remove a built-in plugin
xterraci build --without cost

# List available built-in plugins
xterraci list-plugins
```

See [`examples/external-plugin/`](examples/external-plugin/) for a minimal plugin example.

## Documentation

Full documentation is available at **[edelwud.github.io/terraci](https://edelwud.github.io/terraci)**.

| Topic | Link |
|-------|------|
| Getting Started | [guide/getting-started](https://edelwud.github.io/terraci/guide/getting-started) |
| Project Structure | [guide/project-structure](https://edelwud.github.io/terraci/guide/project-structure) |
| Dependencies | [guide/dependencies](https://edelwud.github.io/terraci/guide/dependencies) |
| Pipeline Generation | [guide/pipeline-generation](https://edelwud.github.io/terraci/guide/pipeline-generation) |
| Configuration Reference | [config/](https://edelwud.github.io/terraci/config/) |
| GitLab MR Integration | [config/gitlab-mr](https://edelwud.github.io/terraci/config/gitlab-mr) |
| Cost Estimation | [config/cost](https://edelwud.github.io/terraci/config/cost) |
| Policy Checks | [config/policy](https://edelwud.github.io/terraci/config/policy) |
| CLI Reference | [cli/](https://edelwud.github.io/terraci/cli/) |

## Contributing

Contributions are welcome! Please open an issue or pull request.

## License

[MIT](LICENSE)
