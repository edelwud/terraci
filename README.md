<p align="center">
  <img src="docs/public/logo.svg" alt="TerraCi" width="120" height="120">
</p>

<h1 align="center">TerraCi</h1>

<p align="center">
  <strong>Terraform Pipeline Generator for GitLab CI</strong><br>
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

TerraCi solves all of this. Point it at your repo, and it generates correct, dependency-aware GitLab CI pipelines — with cost estimates and policy checks baked in.

## Features

<table>
<tr><td width="50%">

**Pipeline Generation**
- Dependency-aware topological ordering
- Parallel execution of independent modules
- Plan + apply stages with manual approval gates
- Changed-only mode via git diff

</td><td>

**Intelligence**
- AWS cost estimation on every plan
- OPA policy enforcement (local, git, OCI sources)
- MR comments with plan summaries, costs & policy results
- Dependency graph visualization (DOT/GraphViz)

</td></tr>
<tr><td>

**Flexibility**
- Terraform & OpenTofu support
- Configurable directory patterns & depth
- Per-job overrides (image, secrets, tags, rules)
- OIDC tokens & Vault secrets integration

</td><td>

**Developer Experience**
- Single binary, zero dependencies
- JSON schema for IDE autocomplete
- Shell completions (bash, zsh, fish)
- Dry-run mode to preview changes

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
# Initialize config
terraci init

# Validate structure & dependencies
terraci validate

# Generate pipeline
terraci generate -o .gitlab-ci.yml

# Only changed modules (CI)
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

## How It Works

```
 Your Repo                    TerraCi                         GitLab CI
┌─────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ platform/   │    │ 1. Discover modules      │    │ deploy-plan-0:      │
│  prod/      │───>│ 2. Parse remote_state    │───>│   plan-vpc          │
│   eu-west/  │    │ 3. Build dependency DAG  │    │ deploy-apply-0:     │
│    vpc/     │    │ 4. Topological sort      │    │   apply-vpc         │
│    eks/     │    │ 5. Generate YAML         │    │ deploy-plan-1:      │
│    rds/     │    │                          │    │   plan-eks, plan-rds │
└─────────────┘    └──────────────────────────┘    └─────────────────────┘
```

### 1. Module Discovery

Scans directories matching `{service}/{environment}/{region}/{module}`:

```
platform/prod/eu-central-1/vpc/   → Module ID: platform/prod/eu-central-1/vpc
platform/prod/eu-central-1/eks/   → Module ID: platform/prod/eu-central-1/eks
platform/prod/eu-central-1/ec2/rabbitmq/  → Submodule (depth 5)
```

### 2. Dependency Resolution

Dependencies are extracted from `terraform_remote_state` data sources:

```hcl
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = "platform/prod/eu-central-1/vpc/terraform.tfstate"
  }
}
# TerraCi resolves: eks → depends on → vpc
```

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
  allow_submodules: true

exclude:
  - "*/test/*"
  - "*/sandbox/*"

gitlab:
  image: "hashicorp/terraform:1.6"    # or "ghcr.io/opentofu/opentofu:1.6"
  terraform_binary: "terraform"        # or "tofu"
  auto_approve: false
  cache_enabled: true

  job_defaults:
    tags: [terraform, docker]
    variables:
      TF_IN_AUTOMATION: "true"

  mr:
    comment:
      enabled: true
    summary_job:
      image:
        name: "ghcr.io/edelwud/terraci:latest"

# Cost estimation (AWS)
cost:
  enabled: true
  show_in_comment: true

# OPA policy checks
policy:
  enabled: true
  sources:
    - path: policies
    - git: https://github.com/org/policies.git
      ref: main
  namespaces: [terraform]
  on_failure: block                    # block, warn, ignore
```

> **Tip:** Run `terraci schema` to generate a JSON schema for IDE autocomplete in your `.terraci.yaml`.

## Commands

| Command | Description |
|---------|-------------|
| `terraci init` | Create `.terraci.yaml` config |
| `terraci validate` | Validate project structure and dependencies |
| `terraci generate` | Generate GitLab CI pipeline |
| `terraci graph` | Visualize dependency graph (DOT, levels) |
| `terraci summary` | Post plan/cost/policy summary to MR (CI) |
| `terraci policy pull` | Download policies from configured sources |
| `terraci policy check` | Evaluate plans against OPA policies |
| `terraci schema` | Generate JSON schema for config validation |
| `terraci version` | Show version and embedded OPA version |

<details>
<summary><strong>Common usage examples</strong></summary>

```bash
# Changed modules only (for MR pipelines)
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml

# Filter by environment or service
terraci generate --environment prod --service platform

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
```

</details>

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
| Policy Checks | [config/policy](https://edelwud.github.io/terraci/config/policy) |
| CLI Reference | [cli/](https://edelwud.github.io/terraci/cli/) |

## Contributing

Contributions are welcome! Please open an issue or pull request.

## License

[MIT](LICENSE)
