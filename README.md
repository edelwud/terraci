<p align="center">
  <img src="docs/public/logo.svg" alt="TerraCi" width="120" height="120">
</p>

<h1 align="center">TerraCi</h1>

<p align="center">
  <strong>Terraform Pipeline Generator for GitLab CI</strong><br>
  Automatically generate pipelines with proper dependency ordering for Terraform/OpenTofu monorepos
</p>

<p align="center">
  <a href="https://github.com/edelwud/terraci/releases"><img src="https://img.shields.io/github/v/release/edelwud/terraci" alt="Release"></a>
  <a href="https://github.com/edelwud/terraci/actions"><img src="https://github.com/edelwud/terraci/actions/workflows/build.yml/badge.svg" alt="Build"></a>
  <a href="https://github.com/edelwud/terraci/blob/main/LICENSE"><img src="https://img.shields.io/github/license/edelwud/terraci" alt="License"></a>
</p>

<p align="center">
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#how-it-works">How It Works</a> •
  <a href="#documentation">Documentation</a>
</p>

---

## Why TerraCi?

Managing Terraform in a monorepo is hard:
- **Dependencies** between modules must be respected (EKS needs VPC first)
- **Manual pipelines** are tedious to write and maintain
- **Full deployments** waste time when only one module changed

TerraCi solves this by analyzing your Terraform code and generating optimal GitLab CI pipelines automatically.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install edelwud/tap/terraci
```

### Go

```bash
go install github.com/edelwud/terraci/cmd/terraci@latest
```

### Docker

```bash
docker run --rm -v $(pwd):/workspace ghcr.io/edelwud/terraci:latest generate
```

### Binary

Download from [GitHub Releases](https://github.com/edelwud/terraci/releases).

## Quick Start

```bash
# 1. Initialize configuration
terraci init

# 2. Validate project structure and dependencies
terraci validate

# 3. Generate GitLab CI pipeline
terraci generate -o .gitlab-ci.yml
```

## How It Works

### 1. Module Discovery

TerraCi scans your directory structure following the pattern `{service}/{environment}/{region}/{module}`:

```
infrastructure/
├── platform/
│   └── prod/
│       └── eu-central-1/
│           ├── vpc/          ← Module: platform/prod/eu-central-1/vpc
│           ├── eks/          ← Module: platform/prod/eu-central-1/eks
│           └── rds/          ← Module: platform/prod/eu-central-1/rds
```

### 2. Dependency Extraction

Dependencies are extracted from `terraform_remote_state` data sources:

```hcl
# eks/main.tf
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = "platform/prod/eu-central-1/vpc/terraform.tfstate"
  }
}
```

TerraCi understands that `eks` depends on `vpc`.

### 3. Pipeline Generation

Modules are sorted topologically and grouped into parallel execution levels:

```
Level 0: vpc (no dependencies)
Level 1: eks, rds (depend on vpc, run in parallel)
Level 2: app (depends on eks)
```

Generated pipeline:

```yaml
stages:
  - deploy-plan-0
  - deploy-apply-0
  - deploy-plan-1
  - deploy-apply-1

plan-vpc:
  stage: deploy-plan-0

apply-vpc:
  stage: deploy-apply-0
  needs: [plan-vpc]

plan-eks:
  stage: deploy-plan-1
  needs: [apply-vpc]
```

## Features

| Feature | Description |
|---------|-------------|
| **Module Discovery** | Pattern-based detection at configurable depth (4-5 levels) |
| **Dependency Graph** | Builds DAG from `terraform_remote_state` references |
| **Topological Sort** | Kahn's algorithm ensures correct execution order |
| **Parallel Execution** | Independent modules run simultaneously |
| **Changed-Only Mode** | Git diff detection for incremental deployments |
| **Policy Checks** | OPA-based policy enforcement on Terraform plans |
| **MR Integration** | Posts plan summaries as GitLab MR comments |
| **OpenTofu Support** | Single config option to switch from Terraform |
| **Visualization** | Export dependency graph to DOT/GraphViz format |

## Commands

| Command | Description |
|---------|-------------|
| `terraci init` | Create configuration file |
| `terraci validate` | Validate structure and dependencies |
| `terraci generate` | Generate GitLab CI pipeline |
| `terraci graph` | Visualize dependency graph |
| `terraci summary` | Post plan results to MR (CI only) |
| `terraci policy pull` | Download policies from configured sources |
| `terraci policy check` | Check Terraform plans against OPA policies |

### Common Options

```bash
# Generate only for changed modules
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml

# Filter by environment
terraci generate --environment prod

# Exclude patterns
terraci generate --exclude "*/sandbox/*" --exclude "*/test/*"

# Auto-approve applies (skip manual trigger)
terraci generate --auto-approve

# Dry run - show what would be generated
terraci generate --dry-run
```

## Configuration

Create `.terraci.yaml` in your project root:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

exclude:
  - "*/test/*"
  - "*/sandbox/*"

gitlab:
  image: "hashicorp/terraform:1.6"
  plan_enabled: true
  auto_approve: false
  cache_enabled: true

  variables:
    TF_IN_AUTOMATION: "true"

  job_defaults:
    tags:
      - terraform
      - docker

  # MR integration
  mr:
    comment:
      enabled: true
    summary_job:
      image:
        name: "ghcr.io/edelwud/terraci:latest"
```

### OpenTofu

```yaml
gitlab:
  terraform_binary: "tofu"
  image: "ghcr.io/opentofu/opentofu:1.6"
```

### Policy Checks

TerraCi integrates [Open Policy Agent (OPA)](https://www.openpolicyagent.org/) to enforce compliance rules:

```yaml
policy:
  enabled: true
  sources:
    - path: policies           # Local policies
    - git: https://github.com/org/policies.git
      ref: main               # Git repository
  namespaces:
    - terraform
  on_failure: block           # block, warn, or ignore
```

Example Rego policy:

```rego
package terraform

deny contains msg if {
    resource := input.resource_changes[_]
    resource.type == "aws_s3_bucket"
    resource.change.after.acl == "public-read"
    msg := sprintf("S3 bucket '%s' must not be public", [resource.name])
}
```

See [examples/policy-checks](examples/policy-checks/) for complete examples.

## Documentation

Full documentation: [terraci.dev](https://terraci.dev) (or `docs/` directory)

- [Getting Started](docs/guide/getting-started.md)
- [Project Structure](docs/guide/project-structure.md)
- [Dependencies](docs/guide/dependencies.md)
- [Configuration Reference](docs/config/index.md)
- [GitLab MR Integration](docs/config/gitlab-mr.md)

## Contributing

Contributions are welcome! Please open an issue or pull request.

## License

MIT
