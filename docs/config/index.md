---
title: Configuration Overview
description: "TerraCi configuration reference: .terraci.yaml file format and default values"
outline: deep
---

# Configuration Overview

TerraCi is configured via a YAML file, typically `.terraci.yaml` in your project root.

## Configuration File

TerraCi looks for configuration in these locations (in order):

1. `.terraci.yaml`
2. `.terraci.yml`
3. `terraci.yaml`
4. `terraci.yml`

Or specify a custom path:

```bash
terraci -c /path/to/config.yaml generate
```

## Quick Start

Initialize a configuration file:

```bash
terraci init
```

This launches an interactive TUI wizard that guides you through provider selection, binary choice, and directory pattern setup. Use `terraci init --ci` for non-interactive mode.

## Full Example

```yaml
# Directory structure configuration
structure:
  pattern: "{service}/{environment}/{region}/{module}"

# Module filtering
exclude:
  - "*/test/*"
  - "*/sandbox/*"
  - "*/.terraform/*"

include: []  # Empty means all (after excludes)

plugins:
  # GitLab CI pipeline settings (used when GITLAB_CI is detected)
  gitlab:
    terraform_binary: "terraform"
    image: "hashicorp/terraform:1.6"
    stages_prefix: "deploy"
    parallelism: 5
    plan_enabled: true
    auto_approve: false
    init_enabled: true

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
      artifacts:
        paths:
          - "*.tfplan"
        expire_in: "1 day"

  # GitHub Actions pipeline settings (used when GITHUB_ACTIONS is detected)
  # github:
  #   terraform_binary: "terraform"
  #   runs_on: "ubuntu-latest"
  #   plan_enabled: true
  #   auto_approve: false
  #   init_enabled: true
  #   permissions:
  #     contents: read
  #     pull-requests: write

  # Summary plugin settings
  # summary:
  #   on_changes_only: false
  #   include_details: true

  # Dependency update checks
  # tfupdate:
  #   enabled: true
  #   policy:
  #     bump: minor
```

## Sections

| Section | Description |
|---------|-------------|
| [structure](./structure) | Directory structure and module discovery |
| [gitlab](./gitlab) | GitLab CI pipeline settings |
| [github](./github) | GitHub Actions pipeline settings |
| [filters](./filters) | Include/exclude patterns |
| [policy](./policy) | OPA policy checks configuration |
| [cost](./cost) | AWS cost estimation configuration |
| [summary](./summary) | Summary plugin |
| [tfupdate](./tfupdate) | Terraform dependency resolution and lock sync |
| [gitlab-mr](./gitlab-mr) | Merge request integration |

## Default Values

If a configuration file is not found, these defaults are used:

```yaml
# provider is auto-detected from CI environment variables:
#   GITHUB_ACTIONS → github
#   GITLAB_CI / CI_SERVER_URL → gitlab
#   fallback → gitlab

structure:
  pattern: "{service}/{environment}/{region}/{module}"

plugins:
  gitlab:
    terraform_binary: "terraform"
    image: "hashicorp/terraform:1.6"
    stages_prefix: "deploy"
    parallelism: 5
    plan_enabled: true
    auto_approve: false
    init_enabled: true
```

## Validation

Validate your configuration:

```bash
terraci validate
```

This checks:
- Required fields are present
- Pattern is parseable
- Image is specified

## Environment Variables

Some values can be overridden via environment variables in the CI pipeline:

```yaml
plugins:
  gitlab:
    variables:
      AWS_REGION: "${AWS_REGION}"  # From CI environment
```

## YAML Anchors

Use YAML anchors for repeated values:

```yaml
defaults: &defaults
  tags:
    - terraform
    - docker
  before_script:
    - aws sts get-caller-identity

plugins:
  gitlab:
    image: "hashicorp/terraform:1.6"
    job_defaults:
      <<: *defaults
```

## OpenTofu with Minimal Images

For OpenTofu minimal images that have a non-shell entrypoint, use the object format:

```yaml
plugins:
  gitlab:
    terraform_binary: "tofu"
    image:
      name: "ghcr.io/opentofu/opentofu:1.9-minimal"
      entrypoint: [""]
```
