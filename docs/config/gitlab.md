# GitLab CI Configuration

The `gitlab` section configures the generated GitLab CI pipeline.

## Options

### terraform_binary

**Type:** `string`
**Default:** `"terraform"`

The Terraform/OpenTofu binary to use.

```yaml
gitlab:
  terraform_binary: "terraform"  # or "tofu"
```

This sets the `TERRAFORM_BINARY` variable in the pipeline.

### image

**Type:** `string` or `object`
**Default:** `"hashicorp/terraform:1.6"`
**Required:** Yes

Docker image for Terraform jobs (in `default` section). Supports both simple string format and object format with entrypoint override.

**String format** (simple):
```yaml
gitlab:
  # Terraform
  image: "hashicorp/terraform:1.6"

  # OpenTofu
  image: "ghcr.io/opentofu/opentofu:1.6"

  # Custom image
  image: "registry.example.com/terraform:1.6"
```

**Object format** (with entrypoint):
```yaml
gitlab:
  # OpenTofu minimal image requires entrypoint override
  image:
    name: "ghcr.io/opentofu/opentofu:1.9-minimal"
    entrypoint: [""]

  # Custom image with specific entrypoint
  image:
    name: "registry.example.com/terraform:1.6"
    entrypoint: ["/bin/sh", "-c"]
```

::: tip OpenTofu Minimal Images
OpenTofu minimal images (e.g., `opentofu:1.9-minimal`) have a non-shell entrypoint. Use the object format with `entrypoint: [""]` to override it for GitLab CI compatibility.
:::

::: warning Deprecation Notice
The `terraform_image` field is deprecated. Use `image` instead.
:::

### stages_prefix

**Type:** `string`
**Default:** `"deploy"`

Prefix for generated stage names.

```yaml
gitlab:
  stages_prefix: "deploy"  # Produces: deploy-plan-0, deploy-apply-0
  # stages_prefix: "terraform"  # Produces: terraform-plan-0, terraform-apply-0
```

### parallelism

**Type:** `integer`
**Default:** `5`

Maximum number of parallel jobs per stage (reserved for future use).

```yaml
gitlab:
  parallelism: 5
```

### plan_enabled

**Type:** `boolean`
**Default:** `true`

Generate separate plan jobs.

```yaml
gitlab:
  plan_enabled: true   # plan + apply jobs
  # plan_enabled: false  # apply only
```

When enabled, generates:
- `plan-*` jobs that run `terraform plan`
- `apply-*` jobs that apply the saved plan

When disabled, generates only `apply-*` jobs that run `terraform apply`.

### auto_approve

**Type:** `boolean`
**Default:** `false`

Auto-approve apply jobs without manual trigger.

```yaml
gitlab:
  auto_approve: false  # Apply requires manual trigger (when: manual)
  # auto_approve: true   # Apply runs automatically
```

### cache_enabled

**Type:** `boolean`
**Default:** `false`

Enable caching of `.terraform` directory for each module. This significantly speeds up pipeline execution by reusing downloaded providers and modules.

```yaml
gitlab:
  cache_enabled: true
```

When enabled, each job will have a cache configuration:

```yaml
plan-platform-prod-vpc:
  cache:
    key: platform-prod-us-east-1-vpc
    paths:
      - platform/prod/us-east-1/vpc/.terraform/
```

The cache key is derived from the module path with slashes replaced by dashes.

### init_enabled

**Type:** `boolean`
**Default:** `true`

Automatically run `terraform init` after changing to the module directory. This ensures initialization happens in the correct context.

```yaml
gitlab:
  init_enabled: true   # Adds ${TERRAFORM_BINARY} init after cd
  # init_enabled: false  # Skip automatic init (use job_defaults.before_script instead)
```

The generated script will be:
```yaml
script:
  - cd platform/prod/us-east-1/vpc     # Change to module directory
  - ${TERRAFORM_BINARY} init            # Auto-added when init_enabled: true
  - ${TERRAFORM_BINARY} plan -out=...   # Main command
```

### variables

**Type:** `map[string]string`
**Default:** `{}`

Global pipeline variables.

```yaml
gitlab:
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"
    AWS_DEFAULT_REGION: "us-east-1"
```

### rules

**Type:** `array`
**Default:** `[]`

Workflow rules for conditional pipeline execution. Controls when pipelines are created.

```yaml
gitlab:
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
      when: always
    - if: '$CI_COMMIT_BRANCH == "main"'
      when: always
    - if: '$CI_COMMIT_TAG'
      when: never
    - when: never
```

Generated output:

```yaml
workflow:
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
      when: always
    - if: '$CI_COMMIT_BRANCH == "main"'
      when: always
    - when: never
```

Each rule can have:
- `if` - Condition expression
- `when` - When to run: `always`, `never`, `on_success`, `manual`, `delayed`
- `changes` - File patterns that trigger the rule

### job_defaults

**Type:** `object`
**Default:** `null`

Default settings applied to all generated jobs (both plan and apply). These are applied before `overwrites`, so overwrites can override job_defaults.

Available fields:
- `image` - Docker image for all jobs
- `id_tokens` - OIDC tokens for all jobs
- `secrets` - Secrets for all jobs
- `before_script` - Commands before each job
- `after_script` - Commands after each job
- `artifacts` - Artifacts configuration
- `tags` - Runner tags
- `rules` - Job-level rules
- `variables` - Additional variables

**Example: Common settings for all jobs**
```yaml
gitlab:
  job_defaults:
    tags:
      - terraform
      - docker
    rules:
      - if: '$CI_COMMIT_BRANCH == "main"'
        when: on_success
    variables:
      CUSTOM_VAR: "value"
```

### overwrites

**Type:** `array`
**Default:** `[]`

Job-level overrides for plan or apply jobs. Allows customizing specific job types with different settings. Applied after `job_defaults`.

Each overwrite has:
- `type` - Which jobs to override: `plan` or `apply`
- `image` - Override Docker image
- `id_tokens` - Override OIDC tokens
- `secrets` - Override secrets
- `before_script` - Override before_script
- `after_script` - Override after_script
- `artifacts` - Override artifacts configuration
- `tags` - Override runner tags
- `rules` - Set job-level rules
- `variables` - Override/add variables

**Example: Different images for plan and apply**
```yaml
gitlab:
  image: "hashicorp/terraform:1.6"

  overwrites:
    - type: plan
      image: "custom/terraform-plan:1.6"
      tags:
        - plan-runner

    - type: apply
      image: "custom/terraform-apply:1.6"
      tags:
        - apply-runner
        - production
```

**Example: Add job-level rules for apply jobs**
```yaml
gitlab:
  overwrites:
    - type: apply
      rules:
        - if: '$CI_COMMIT_BRANCH == "main"'
          when: manual
        - when: never
```

**Example: Different secrets for different job types**
```yaml
gitlab:
  job_defaults:
    secrets:
      COMMON_SECRET:
        vault: common/secret@namespace

  overwrites:
    - type: apply
      secrets:
        DEPLOY_KEY:
          vault: deploy/key@namespace
          file: true
```

**Example: job_defaults with overwrites**
```yaml
gitlab:
  # Common settings for all jobs
  job_defaults:
    tags:
      - terraform
    rules:
      - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
        when: on_success

  # Override for apply jobs only
  overwrites:
    - type: apply
      tags:
        - terraform
        - production
      rules:
        - if: '$CI_COMMIT_BRANCH == "main"'
          when: manual
```

## Full Example

```yaml
gitlab:
  # Binary configuration
  terraform_binary: "terraform"
  image: "hashicorp/terraform:1.6"

  # Pipeline structure
  stages_prefix: "deploy"
  parallelism: 5
  plan_enabled: true
  auto_approve: false
  cache_enabled: true
  init_enabled: true

  # Pipeline variables
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"

  # Workflow rules
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
      when: always
    - if: '$CI_COMMIT_BRANCH == "main"'
      when: always

  # Job defaults (applied to all jobs)
  job_defaults:
    tags:
      - terraform
      - docker
    before_script:
      - aws sts get-caller-identity
    after_script:
      - echo "Job completed"
    id_tokens:
      AWS_OIDC_TOKEN:
        aud: "https://gitlab.example.com"
    secrets:
      CREDENTIALS:
        vault: ci/terraform/credentials@namespace
    rules:
      - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
        when: on_success

  # Job overwrites (override job_defaults for specific job types)
  overwrites:
    - type: apply
      tags:
        - production
        - secure
      rules:
        - if: '$CI_COMMIT_BRANCH == "main"'
          when: manual
```

## Generated Output

With the above configuration, TerraCi generates:

```yaml
variables:
  TERRAFORM_BINARY: "terraform"
  TF_IN_AUTOMATION: "true"
  TF_INPUT: "false"

default:
  image: hashicorp/terraform:1.6

workflow:
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
      when: always
    - if: '$CI_COMMIT_BRANCH == "main"'
      when: always

stages:
  - deploy-plan-0
  - deploy-apply-0

plan-platform-prod-vpc:
  stage: deploy-plan-0
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan
  variables:
    TF_MODULE: vpc
    # ...
  tags:
    - terraform
    - docker
  before_script:
    - aws sts get-caller-identity
  after_script:
    - echo "Job completed"
  id_tokens:
    AWS_OIDC_TOKEN:
      aud: "https://gitlab.example.com"
  secrets:
    CREDENTIALS:
      vault: ci/terraform/credentials@namespace
  artifacts:
    paths:
      - platform/prod/us-east-1/vpc/plan.tfplan
    expire_in: 1 day
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
      when: on_success
  cache:
    key: platform-prod-us-east-1-vpc
    paths:
      - platform/prod/us-east-1/vpc/.terraform/

apply-platform-prod-vpc:
  stage: deploy-apply-0
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} apply plan.tfplan
  needs:
    - plan-platform-prod-vpc
  tags:
    - production
    - secure
  before_script:
    - aws sts get-caller-identity
  after_script:
    - echo "Job completed"
  id_tokens:
    AWS_OIDC_TOKEN:
      aud: "https://gitlab.example.com"
  secrets:
    CREDENTIALS:
      vault: ci/terraform/credentials@namespace
  rules:
    - if: '$CI_COMMIT_BRANCH == "main"'
      when: manual
```

## Per-Job Variables

Each job receives these variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `TF_MODULE_PATH` | Relative path to module | `platform/prod/us-east-1/vpc` |
| `TF_SERVICE` | Service name | `platform` |
| `TF_ENVIRONMENT` | Environment name | `prod` |
| `TF_REGION` | Region name | `us-east-1` |
| `TF_MODULE` | Module name | `vpc` |
