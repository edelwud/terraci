---
layout: home

hero:
  name: TerraCi
  text: Terraform Pipeline Generator
  tagline: Generate GitLab CI pipelines with dependency ordering for Terraform/OpenTofu monorepos
  image:
    src: /logo.svg
    alt: TerraCi
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/edelwud/terraci

features:
  - icon:
      src: /icons/search.svg
    title: Module Discovery
    details: Scans directory structure to find Terraform modules. Pattern-based detection with configurable depth (4-5 levels).
    link: /guide/project-structure
    linkText: Learn more
  - icon:
      src: /icons/graph.svg
    title: Dependency Graph
    details: Extracts dependencies from terraform_remote_state blocks. Builds DAG with topological sorting.
    link: /guide/dependencies
    linkText: How it works
  - icon:
      src: /icons/zap.svg
    title: Parallel Execution
    details: Groups modules into execution levels. Independent modules run in parallel, dependent modules wait.
    link: /guide/pipeline-generation
    linkText: See example
  - icon:
      src: /icons/git.svg
    title: Changed-Only Mode
    details: Detects modified files via git diff. Generates pipelines only for affected modules and dependents.
    link: /guide/git-integration
    linkText: Git integration
  - icon:
      src: /icons/tofu.svg
    title: OpenTofu Ready
    details: Supports both Terraform and OpenTofu. Single config option to switch between them.
    link: /guide/opentofu
    linkText: Configure
  - icon:
      src: /icons/chart.svg
    title: Visualization
    details: Export dependency graph to DOT format. Visualize with GraphViz or other tools.
    link: /cli/graph
    linkText: View commands
---

## Install

```bash
# Homebrew (macOS/Linux)
brew install edelwud/tap/terraci

# Go
go install github.com/edelwud/terraci/cmd/terraci@latest

# Docker
docker run --rm -v $(pwd):/workspace ghcr.io/edelwud/terraci:latest generate
```

## Usage

```bash
# Initialize config
terraci init

# Generate pipeline
terraci generate -o .gitlab-ci.yml

# Only changed modules
terraci generate --changed-only --base-ref main
```

## How It Works

**1. Discover modules** from directory structure:

```
platform/prod/eu-central-1/
├── vpc/        → platform/prod/eu-central-1/vpc
├── eks/        → platform/prod/eu-central-1/eks
└── rds/        → platform/prod/eu-central-1/rds
```

**2. Extract dependencies** from `terraform_remote_state`:

```hcl
# eks/main.tf
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = "platform/prod/eu-central-1/vpc/terraform.tfstate"
  }
}
```

**3. Build execution order**:

```
Level 0: vpc (no dependencies)
Level 1: eks, rds (depend on vpc)
```

**4. Generate pipeline**:

```yaml
stages:
  - plan-0
  - apply-0
  - plan-1
  - apply-1

plan-vpc:
  stage: plan-0

apply-vpc:
  stage: apply-0
  needs: [plan-vpc]

plan-eks:
  stage: plan-1
  needs: [apply-vpc]
```

## Configuration

```yaml
# .terraci.yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

gitlab:
  image: hashicorp/terraform:1.6
  plan_enabled: true

exclude:
  - "*/test/*"
```

[Full configuration reference →](/config/)
