---
title: Getting Started
description: Install TerraCi, initialize configuration, and generate your first CI pipeline
outline: deep
---

# Getting Started

This guide will help you set up TerraCi and generate your first pipeline.

## Installation

### Homebrew (Recommended)

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

Download from [GitHub Releases](https://github.com/edelwud/terraci/releases) and add to your PATH.

### From Source

```bash
git clone https://github.com/edelwud/terraci.git
cd terraci
make build
./terraci version
```

## Quick Start

### 1. Initialize Configuration

Navigate to your Terraform project root and run:

```bash
terraci init
```

By default, this launches an interactive TUI wizard that guides you through project setup. For non-interactive mode (e.g., in CI scripts), use:

```bash
terraci init --ci
terraci init --ci --provider github
terraci init --ci --provider gitlab --binary tofu
```

This creates a `.terraci.yaml` configuration file:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

execution:
  binary: terraform
  plan_enabled: true

extensions:
  gitlab:
    image:
      name: hashicorp/terraform:1.6
    auto_approve: false
```

### 2. Validate Your Project

Check that TerraCi correctly discovers your modules:

```bash
terraci validate
```

Expected output:

```
✓ Found 12 modules
✓ Built dependency graph with 15 edges
✓ No circular dependencies detected
✓ 4 execution levels identified

Execution order:
  Level 0: vpc, iam
  Level 1: eks, rds, elasticache
  Level 2: app-backend, app-frontend
  Level 3: monitoring
```

### 3. Visualize Dependencies (Optional)

Export the dependency graph:

```bash
terraci graph --format dot -o deps.dot
dot -Tpng deps.dot -o deps.png
```

### 4. Generate Pipeline

Generate a CI pipeline:

```bash
# GitLab CI
terraci generate -o .gitlab-ci.yml

# GitHub Actions
terraci generate -o .github/workflows/terraform.yml
```

### 5. Generate for Changed Modules Only

For incremental deployments:

```bash
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

::: tip CI Environments
TerraCi resolves the merge-base via go-git, so `--changed-only` works against any pushable base ref. CI runners that clone with `--depth 1` (GitLab and GitHub default) need a deep history to compute a reliable diff — either set the runner's `GIT_DEPTH` (or actions/checkout `fetch-depth: 0`) high enough, or enable auto-unshallow:

```yaml
extensions:
  git:
    auto_unshallow: true   # runs `git fetch` with depth=0 and clears .git/shallow when needed
```
:::

## Project Structure

TerraCi expects your Terraform modules to follow this structure:

```
your-project/
├── .terraci.yaml          # TerraCi configuration
├── service-a/
│   ├── production/
│   │   └── us-east-1/
│   │       ├── vpc/
│   │       │   └── main.tf
│   │       └── eks/
│   │           └── main.tf
│   └── staging/
│       └── us-east-1/
│           └── vpc/
│               └── main.tf
└── service-b/
    └── production/
        └── eu-west-1/
            └── rds/
                └── main.tf
```

The pattern `{service}/{environment}/{region}/{module}` maps to:
- `service-a/production/us-east-1/vpc`
- `service-a/production/us-east-1/eks`
- `service-b/production/eu-west-1/rds`

## Defining Dependencies

TerraCi discovers dependencies from `terraform_remote_state` data sources:

```hcl
# In eks/main.tf
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "my-terraform-state"
    key    = "service-a/production/us-east-1/vpc/terraform.tfstate"
    region = "us-east-1"
  }
}

resource "aws_eks_cluster" "main" {
  vpc_config {
    subnet_ids = data.terraform_remote_state.vpc.outputs.private_subnet_ids
  }
}
```

TerraCi parses the `key` path and matches it to the `vpc` module.

## Next Steps

- [Project Structure](/guide/project-structure) - Learn about supported directory layouts
- [Dependency Resolution](/guide/dependencies) - Understand how dependencies are detected
- [Configuration Reference](/config/) - Full configuration options
- [CLI Reference](/cli/) - All available commands
