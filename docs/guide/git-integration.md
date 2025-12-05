# Git Integration

TerraCi integrates with Git to generate pipelines only for changed modules and their dependents.

## Changed-Only Mode

Enable changed-only mode with the `--changed-only` flag:

```bash
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

## How It Works

### 1. Detect Changed Files

TerraCi runs `git diff` to find changed files:

```bash
git diff --name-only main...HEAD
```

Output:
```
platform/production/us-east-1/vpc/main.tf
platform/production/us-east-1/vpc/variables.tf
shared/modules/vpc/main.tf
```

### 2. Map Files to Modules

Changed files are mapped to their parent modules:

| File | Module |
|------|--------|
| `platform/production/us-east-1/vpc/main.tf` | `platform/production/us-east-1/vpc` |
| `platform/production/us-east-1/vpc/variables.tf` | `platform/production/us-east-1/vpc` |

Files outside module directories (like `shared/modules/`) are ignored.

### 3. Find Affected Modules

TerraCi traverses the dependency graph to find all modules that depend on the changed modules:

```
Changed: vpc
    ↓
Dependents: eks, rds, cache
    ↓
Dependents: app-backend, app-frontend
```

All these modules are included in the pipeline.

### 4. Generate Pipeline

A pipeline is generated only for the affected modules, maintaining proper dependency order.

## Reference Options

### Base Reference

Specify the base branch or commit:

```bash
# Compare against main branch
terraci generate --changed-only --base-ref main

# Compare against specific commit
terraci generate --changed-only --base-ref abc123

# Compare against tag
terraci generate --changed-only --base-ref v1.0.0
```

### Auto-Detect Default Branch

If `--base-ref` is not specified, TerraCi tries to detect the default branch:

```bash
terraci generate --changed-only  # Detects main/master automatically
```

### Compare Commits

Compare between specific commits:

```bash
terraci generate --changed-only --base-ref HEAD~5  # Last 5 commits
```

## Use Cases

### Pull Request Pipelines

Generate a pipeline for changes in a merge request:

```yaml
# .gitlab-ci.yml
generate-pipeline:
  stage: prepare
  script:
    - terraci generate --changed-only --base-ref $CI_MERGE_REQUEST_TARGET_BRANCH_NAME -o generated-pipeline.yml
  artifacts:
    paths:
      - generated-pipeline.yml

deploy:
  stage: deploy
  trigger:
    include:
      - artifact: generated-pipeline.yml
        job: generate-pipeline
```

### Feature Branch Deployments

Deploy only changed infrastructure on feature branches:

```yaml
deploy-feature:
  script:
    - terraci generate --changed-only --base-ref main -o pipeline.yml
    - gitlab-runner exec shell < pipeline.yml
  rules:
    - if: $CI_COMMIT_BRANCH != "main"
```

### Scheduled Full Deployments

Run full deployments on schedule, changed-only otherwise:

```yaml
generate:
  script:
    - |
      if [ "$CI_PIPELINE_SOURCE" = "schedule" ]; then
        terraci generate -o pipeline.yml
      else
        terraci generate --changed-only --base-ref main -o pipeline.yml
      fi
```

## Filtering Combined with Git

Combine git detection with filters:

```bash
# Only production changes
terraci generate --changed-only --base-ref main --environment production

# Exclude test modules from change detection
terraci generate --changed-only --base-ref main --exclude "*/test/*"
```

## Viewing Changed Modules

See which modules would be affected without generating:

```bash
terraci generate --changed-only --base-ref main --dry-run
```

Output:
```
Changed files:
  - platform/production/us-east-1/vpc/main.tf

Changed modules:
  - platform/production/us-east-1/vpc

Affected modules (including dependents):
  - platform/production/us-east-1/vpc
  - platform/production/us-east-1/eks
  - platform/production/us-east-1/rds
  - platform/production/us-east-1/app
```

## Troubleshooting

### No Changes Detected

If no modules are detected as changed:

1. Verify the base reference exists:
   ```bash
   git rev-parse main
   ```

2. Check git diff manually:
   ```bash
   git diff --name-only main...HEAD
   ```

3. Ensure changed files are in module directories

### Too Many Modules Affected

If more modules are affected than expected:

1. Check the dependency graph:
   ```bash
   terraci graph --format list
   ```

2. Verify remote state references are correct

3. Consider if the dependency chain is intentional

### Uncommitted Changes

TerraCi only considers committed changes. To include uncommitted:

```bash
# Stage and commit first
git add -A
git commit -m "WIP"
terraci generate --changed-only --base-ref main
```

Or use the default branch comparison which includes uncommitted changes:

```bash
terraci generate --changed-only  # Includes working directory changes
```
