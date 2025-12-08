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

This creates `.terraci.yaml` with sensible defaults.

## Full Example

```yaml
# Directory structure configuration
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

# Module filtering
exclude:
  - "*/test/*"
  - "*/sandbox/*"
  - "*/.terraform/*"

include: []  # Empty means all (after excludes)

# GitLab CI pipeline settings
gitlab:
  terraform_binary: "terraform"
  terraform_image: "hashicorp/terraform:1.6"
  stages_prefix: "deploy"
  parallelism: 5
  plan_enabled: true
  auto_approve: false

  before_script:
    - ${TERRAFORM_BINARY} init

  after_script: []

  tags:
    - terraform
    - docker

  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"

  artifact_paths:
    - "*.tfplan"

# Backend configuration (for path matching)
backend:
  type: s3
  bucket: my-terraform-state
  region: us-east-1
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

## Sections

| Section | Description |
|---------|-------------|
| [structure](./structure) | Directory structure and module discovery |
| [gitlab](./gitlab) | GitLab CI pipeline settings |
| [filters](./filters) | Include/exclude patterns |

## Default Values

If a configuration file is not found, these defaults are used:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

gitlab:
  terraform_binary: "terraform"
  terraform_image: "hashicorp/terraform:1.6"
  stages_prefix: "deploy"
  parallelism: 5
  plan_enabled: true
  auto_approve: false
  before_script:
    - ${TERRAFORM_BINARY} init
  artifact_paths:
    - "*.tfplan"

backend:
  type: s3
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

## Validation

Validate your configuration:

```bash
terraci validate
```

This checks:
- Required fields are present
- Depth values are valid
- Pattern is parseable
- Image is specified

## Environment Variables

Some values can be overridden via environment variables in the CI pipeline:

```yaml
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
    - ${TERRAFORM_BINARY} init

gitlab:
  <<: *defaults
  terraform_image: "hashicorp/terraform:1.6"
```

## OpenTofu with Minimal Images

For OpenTofu minimal images that have a non-shell entrypoint, use the object format:

```yaml
gitlab:
  terraform_binary: "tofu"
  terraform_image:
    name: "ghcr.io/opentofu/opentofu:1.9-minimal"
    entrypoint: [""]
```
