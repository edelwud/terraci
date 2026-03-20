---
title: terraci generate
description: Generate CI pipelines with dependency ordering and changed-only mode
outline: deep
---

# terraci generate

Generate CI pipeline (GitLab CI or GitHub Actions) from Terraform modules.

## Synopsis

```bash
terraci generate [flags]
```

## Description

The `generate` command scans your project, builds a dependency graph, and outputs a CI pipeline YAML file (GitLab CI or GitHub Actions, depending on the configured provider).

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | stdout | Output file path |
| `--changed-only` | | bool | false | Generate for changed modules only |
| `--base-ref` | | string | auto | Base git ref for change detection |
| `--exclude` | `-x` | string[] | | Exclude patterns |
| `--include` | `-i` | string[] | | Include patterns |
| `--filter` | `-f` | string[] | | Filter by segment (`key=value`, e.g. `environment=prod`) |
| `--plan-only` | | bool | false | Generate only plan jobs (no apply) |
| `--auto-approve` | | bool | false | Auto-approve apply jobs |
| `--no-auto-approve` | | bool | false | Require manual trigger for apply |
| `--dry-run` | | bool | false | Preview without output |

## Examples

### Basic Generation

```bash
# GitLab CI output
terraci generate -o .gitlab-ci.yml

# GitHub Actions output
terraci generate -o .github/workflows/terraform.yml

# Output to stdout
terraci generate
```

The provider is auto-detected from CI environment variables, or set via `provider:` in `.terraci.yaml`.

### Changed-Only Mode

```bash
# Detect changes from main branch
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml

# Detect changes from specific commit
terraci generate --changed-only --base-ref abc123 -o .gitlab-ci.yml

# Auto-detect default branch
terraci generate --changed-only -o .gitlab-ci.yml
```

### Filtering

```bash
# By environment
terraci generate --filter environment=production -o .gitlab-ci.yml

# By service
terraci generate --filter service=platform -o .gitlab-ci.yml

# By region
terraci generate --filter region=us-east-1 -o .gitlab-ci.yml

# Combined filters (AND across different keys)
terraci generate --filter service=platform --filter environment=production --filter region=us-east-1 -o .gitlab-ci.yml

# Multiple values for one key (OR within same key)
terraci generate --filter environment=stage --filter environment=prod -o .gitlab-ci.yml
```

The `--filter` flag works with any segment name defined in your `structure.pattern`.

### Exclude/Include Patterns

```bash
# Exclude test modules
terraci generate --exclude "*/test/*" -o .gitlab-ci.yml

# Multiple excludes
terraci generate -x "*/test/*" -x "*/sandbox/*" -o .gitlab-ci.yml

# Include specific pattern
terraci generate --include "platform/*/*/*" -o .gitlab-ci.yml
```

### Dry Run

```bash
terraci generate --dry-run
```

Output:
```
Dry Run Summary:
  Total modules: 15
  Affected modules: 8
  Stages: 6
  Jobs: 16

Execution Order:
  Level 0: [vpc, iam]
  Level 1: [eks, rds, cache]
  Level 2: [app-backend, app-frontend]
  Level 3: [monitoring]
```

### Pipe to Tools

```bash
# Extract stages
terraci generate | yq '.stages'

# Validate syntax
terraci generate | gitlab-ci-lint

# Diff with current
terraci generate > new.yml && diff .gitlab-ci.yml new.yml
```

## Output Structure

### GitLab CI Output

```yaml
# Global variables
variables:
  TERRAFORM_BINARY: "terraform"

# Default job settings
default:
  image: hashicorp/terraform:1.6

# Stages for each execution level
stages:
  - deploy-plan-0
  - deploy-apply-0
  - deploy-plan-1
  - deploy-apply-1

# Plan jobs
plan-service-env-region-module:
  stage: deploy-plan-0
  script:
    - cd service/env/region/module
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan

# Apply jobs
apply-service-env-region-module:
  stage: deploy-apply-0
  needs:
    - plan-service-env-region-module
  script:
    - cd service/env/region/module
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} apply plan.tfplan
```

### GitHub Actions Output

```yaml
name: Terraform
on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read
  pull-requests: write

jobs:
  plan-service-env-region-module:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: hashicorp/setup-terraform@v3
      - name: Terraform Plan
        run: |
          cd service/env/region/module
          terraform init
          terraform plan -out=plan.tfplan
    env:
      TF_MODULE_PATH: service/env/region/module
      TF_SERVICE: service
      TF_ENVIRONMENT: env
      TF_REGION: region
      TF_MODULE: module
```

## Use Cases

### CI/CD Integration

```yaml
# .gitlab-ci.yml
stages:
  - prepare
  - deploy

generate-pipeline:
  stage: prepare
  script:
    - terraci generate --changed-only --base-ref $CI_MERGE_REQUEST_TARGET_BRANCH_NAME -o pipeline.yml
  artifacts:
    paths:
      - pipeline.yml

trigger-deploy:
  stage: deploy
  trigger:
    include:
      - artifact: pipeline.yml
        job: generate-pipeline
```

### Scheduled Full Deploy

```yaml
deploy:
  script:
    - |
      if [ "$CI_PIPELINE_SOURCE" = "schedule" ]; then
        terraci generate -o pipeline.yml
      else
        terraci generate --changed-only -o pipeline.yml
      fi
```

### Environment-Specific Pipelines

```bash
# Generate for each environment separately
terraci generate --filter environment=production -o production.yml
terraci generate --filter environment=staging -o staging.yml
```

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| No modules found | Wrong depth or missing .tf files | Check structure config |
| Circular dependency | Modules depend on each other cyclically | Fix remote_state references |
| Git ref not found | Invalid base-ref | Verify branch/commit exists |

## See Also

- [Pipeline Generation Guide](/guide/pipeline-generation) — end-to-end guide for generating CI pipelines
- [GitLab CI Configuration](/config/gitlab) — GitLab pipeline generation settings including images, stages, and jobs
- [GitHub Actions Configuration](/config/github) — GitHub Actions workflow settings including runners, steps, and permissions
