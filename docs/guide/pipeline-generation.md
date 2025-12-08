# Pipeline Generation

TerraCi generates GitLab CI pipelines that respect module dependencies and enable parallel execution.

## Basic Generation

Generate a pipeline for all modules:

```bash
terraci generate -o .gitlab-ci.yml
```

## Pipeline Structure

The generated pipeline includes:

### Stages

Stages are created for each execution level:

```yaml
stages:
  - deploy-plan-0    # Plan for level 0 modules
  - deploy-apply-0   # Apply for level 0 modules
  - deploy-plan-1    # Plan for level 1 modules
  - deploy-apply-1   # Apply for level 1 modules
```

### Variables

Global variables are set from configuration:

```yaml
variables:
  TERRAFORM_BINARY: "terraform"
  TF_IN_AUTOMATION: "true"
  TF_INPUT: "false"
```

### Default Configuration

Shared job settings:

```yaml
default:
  image: hashicorp/terraform:1.6
  before_script:
    - ${TERRAFORM_BINARY} init
  tags:
    - terraform
    - docker
```

### Jobs

Two jobs per module (if `plan_enabled: true`):

```yaml
plan-platform-prod-us-east-1-vpc:
  stage: deploy-plan-0
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan
  variables:
    TF_MODULE_PATH: platform/prod/us-east-1/vpc
    TF_SERVICE: platform
    TF_ENVIRONMENT: prod
    TF_REGION: us-east-1
    TF_MODULE: vpc
  artifacts:
    paths:
      - platform/prod/us-east-1/vpc/plan.tfplan
    expire_in: 1 day
  resource_group: platform/prod/us-east-1/vpc

apply-platform-prod-us-east-1-vpc:
  stage: deploy-apply-0
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} apply plan.tfplan
  needs:
    - job: plan-platform-prod-us-east-1-vpc
  when: manual
  resource_group: platform/prod/us-east-1/vpc
```

## Job Dependencies

Jobs use GitLab's `needs` keyword to express dependencies:

```yaml
plan-platform-prod-us-east-1-eks:
  stage: deploy-plan-1
  needs:
    - job: apply-platform-prod-us-east-1-vpc  # Wait for VPC
  # ...

apply-platform-prod-us-east-1-eks:
  stage: deploy-apply-1
  needs:
    - job: plan-platform-prod-us-east-1-eks   # Wait for own plan
    - job: apply-platform-prod-us-east-1-vpc  # Wait for VPC
  # ...
```

## Parallel Execution

Independent modules at the same level run in parallel:

```
Level 0: vpc ──────┬──────────────────────────────────→
         iam ──────┘
                   ↓
Level 1: eks ──────┬──────────────────────────────────→
         rds ──────┤
         cache ────┘
                   ↓
Level 2: app ─────────────────────────────────────────→
```

## Changed-Only Pipelines

Generate pipelines for changed modules and their dependents:

```bash
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

This:
1. Detects files changed since `main` branch
2. Maps changed files to modules
3. Finds all modules that depend on changed modules
4. Generates a pipeline only for affected modules

### Example

If `vpc/main.tf` changes:
- `vpc` is included (changed)
- `eks` is included (depends on vpc)
- `rds` is included (depends on vpc)
- `app` is included (depends on eks and rds)

Unchanged modules like `monitoring` (no vpc dependency) are excluded.

## Resource Groups

Each module uses a `resource_group` to prevent concurrent applies:

```yaml
apply-platform-prod-us-east-1-vpc:
  resource_group: platform/prod/us-east-1/vpc
```

This ensures only one apply job runs per module at a time.

## Configuration Options

### Plan Stage

Enable or disable the plan stage:

```yaml
gitlab:
  plan_enabled: true  # Generate plan jobs
  # plan_enabled: false  # Skip straight to apply
```

### Auto-Approve

Skip manual approval for apply jobs:

```yaml
gitlab:
  auto_approve: false  # Require manual trigger (default)
  # auto_approve: true  # Run apply automatically
```

You can also override this via CLI:

```bash
# Enable auto-approve (skip manual trigger)
terraci generate --auto-approve -o .gitlab-ci.yml

# Disable auto-approve (require manual trigger)
terraci generate --no-auto-approve -o .gitlab-ci.yml
```

CLI flags override the configuration file setting.

### Stage Prefix

Customize stage names:

```yaml
gitlab:
  stages_prefix: "terraform"  # terraform-plan-0, terraform-apply-0
```

### Custom Scripts

Add custom before/after scripts via `job_defaults`:

```yaml
gitlab:
  job_defaults:
    before_script:
      - ${TERRAFORM_BINARY} init
      - ${TERRAFORM_BINARY} workspace select ${TF_ENVIRONMENT} || ${TERRAFORM_BINARY} workspace new ${TF_ENVIRONMENT}
    after_script:
      - ${TERRAFORM_BINARY} output -json > outputs.json
```

### Runner Tags

Specify GitLab runner tags via `job_defaults`:

```yaml
gitlab:
  job_defaults:
    tags:
      - terraform
      - docker
      - aws
```

## Dry Run

Preview what would be generated:

```bash
terraci generate --dry-run
```

Output:
```
Dry Run Summary:
  Total modules: 12
  Affected modules: 5
  Stages: 6
  Jobs: 10

Execution Order:
  Level 0: [vpc]
  Level 1: [eks, rds]
  Level 2: [app-backend, app-frontend]
```

## Output Formats

### File Output

```bash
terraci generate -o .gitlab-ci.yml
```

### Stdout

```bash
terraci generate  # Prints to stdout
```

### Pipe to Other Tools

```bash
terraci generate | yq '.stages'  # Extract stages
```
