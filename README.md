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
├── cdp/
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
  terraform_image: "hashicorp/terraform:1.6"
  parallelism: 5
  plan_enabled: true
```

## Examples

### Dependency graph in DOT format

```bash
terraci graph --format dot -o deps.dot
dot -Tpng deps.dot -o deps.png
```

### Filter by environment

```bash
terraci generate --environment prod -o prod-pipeline.yml
```

### Exclude modules

```bash
terraci generate --exclude "*/sandbox/*" --exclude "*/test/*"
```

## License

MIT
