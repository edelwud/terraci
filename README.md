<p align="center">
  <img src="docs/public/logo.svg" alt="TerraCi" width="120" height="120">
</p>

<h1 align="center">TerraCi</h1>

<p align="center">
  <strong>Terraform Pipeline Generator</strong><br>
  Automatically generate GitLab CI pipelines with proper dependency ordering for your Terraform/OpenTofu monorepos
</p>

<p align="center">
  <a href="https://github.com/edelwud/terraci/releases"><img src="https://img.shields.io/github/v/release/edelwud/terraci" alt="Release"></a>
  <a href="https://github.com/edelwud/terraci/actions"><img src="https://github.com/edelwud/terraci/actions/workflows/build.yml/badge.svg" alt="Build"></a>
  <a href="https://github.com/edelwud/terraci/blob/main/LICENSE"><img src="https://img.shields.io/github/license/edelwud/terraci" alt="License"></a>
</p>

---

## Features

- Automatic discovery of Terraform modules based on directory structure
- Dependency extraction from `terraform_remote_state` (including `for_each`)
- Dependency graph construction with topological sorting
- GitLab CI pipeline generation with correct execution order
- Module filtering using glob patterns
- Git integration: generate pipelines only for changed modules
- Support for Terraform and OpenTofu
- Caching of `.terraform` directory for faster pipelines
- OIDC tokens and Vault secrets integration

## Installation

```bash
# From source
go install github.com/edelwud/terraci/cmd/terraci@latest

# Or build locally
make build
```

## Quick Start

```bash
# Initialize configuration
terraci init

# Validate project structure
terraci validate

# Generate pipeline
terraci generate -o .gitlab-ci.yml

# Only for changed modules
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml

# Auto-approve applies (skip manual trigger)
terraci generate --auto-approve -o .gitlab-ci.yml
```

## Project Structure

TerraCi expects the following directory structure:

```
project/
├── service/
│   └── environment/
│       └── region/
│           ├── module/           # depth 4
│           │   └── main.tf
│           └── module/
│               └── submodule/    # depth 5 (optional)
│                   └── main.tf
```

Example:
```
infrastructure/
├── platform/
│   ├── stage/
│   │   └── eu-central-1/
│   │       ├── vpc/
│   │       ├── eks/
│   │       └── ec2/
│   │           └── rabbitmq/    # submodule
│   └── prod/
│       └── eu-central-1/
│           └── vpc/
```

## Commands

| Command | Description |
|---------|-------------|
| `terraci generate` | Generate GitLab CI pipeline |
| `terraci validate` | Validate structure and dependencies |
| `terraci graph` | Visualize dependency graph |
| `terraci init` | Create configuration file |
| `terraci version` | Show version information |

### Generate Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output file (default: stdout) |
| `--changed-only` | Only include changed modules and dependents |
| `--base-ref` | Base git ref for change detection |
| `--auto-approve` | Auto-approve apply jobs (skip manual trigger) |
| `--no-auto-approve` | Require manual trigger for apply jobs |
| `-x, --exclude` | Glob patterns to exclude modules |
| `-i, --include` | Glob patterns to include modules |
| `-s, --service` | Filter by service name |
| `-e, --environment` | Filter by environment |
| `-r, --region` | Filter by region |
| `--dry-run` | Show what would be generated |

## Configuration

File `.terraci.yaml`:

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
  # Binary and image
  terraform_binary: "terraform"
  image: "hashicorp/terraform:1.6"

  # Pipeline settings
  stages_prefix: "deploy"
  plan_enabled: true
  auto_approve: false
  cache_enabled: true
  init_enabled: true

  # Global variables
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"

  # Job defaults (applied to all jobs)
  job_defaults:
    tags:
      - terraform
      - docker
    before_script:
      - aws sts get-caller-identity

  # Overwrites for specific job types
  overwrites:
    - type: apply
      tags:
        - production
      rules:
        - if: '$CI_COMMIT_BRANCH == "main"'
          when: manual

backend:
  type: s3
  bucket: my-terraform-state
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

### OpenTofu Support

```yaml
gitlab:
  terraform_binary: "tofu"
  image: "ghcr.io/opentofu/opentofu:1.6"
```

For minimal images requiring entrypoint override:

```yaml
gitlab:
  terraform_binary: "tofu"
  image:
    name: "ghcr.io/opentofu/opentofu:1.9-minimal"
    entrypoint: [""]
```

## Examples

### Changed modules only

```bash
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

### Auto-approve applies

```bash
# Enable auto-approve (skip manual trigger)
terraci generate --auto-approve -o .gitlab-ci.yml

# Disable auto-approve (override config)
terraci generate --no-auto-approve -o .gitlab-ci.yml
```

### Filter by environment

```bash
terraci generate --environment prod -o prod-pipeline.yml
```

### Exclude modules

```bash
terraci generate --exclude "*/sandbox/*" --exclude "*/test/*"
```

### Dependency graph

```bash
# DOT format for visualization
terraci graph --format dot -o deps.dot
dot -Tpng deps.dot -o deps.png

# Show execution levels
terraci graph --format levels

# Show dependents of a module
terraci graph --module platform/stage/eu-central-1/vpc --dependents
```

### Dry run

```bash
terraci generate --dry-run
```

Output:
```
Dry Run Results:
  Total modules discovered: 12
  Modules to process: 5
  Pipeline stages: 6
  Pipeline jobs: 10

Execution order:
  Level 0: [vpc]
  Level 1: [eks, rds]
  Level 2: [app]
```

## Documentation

Full documentation available at [terraci.dev](https://terraci.dev) (if deployed) or in the `docs/` directory.

## License

MIT
