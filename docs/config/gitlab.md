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

### terraform_image

**Type:** `string`
**Default:** `"hashicorp/terraform:1.6"`
**Required:** Yes

Docker image for Terraform jobs.

```yaml
gitlab:
  # Terraform
  terraform_image: "hashicorp/terraform:1.6"

  # OpenTofu
  terraform_image: "ghcr.io/opentofu/opentofu:1.6"

  # Custom image
  terraform_image: "registry.example.com/terraform:1.6"
```

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

### before_script

**Type:** `string[]`
**Default:** `["${TERRAFORM_BINARY} init"]`

Commands to run before each job.

```yaml
gitlab:
  before_script:
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} workspace select ${TF_ENVIRONMENT} || ${TERRAFORM_BINARY} workspace new ${TF_ENVIRONMENT}
```

### after_script

**Type:** `string[]`
**Default:** `[]`

Commands to run after each job.

```yaml
gitlab:
  after_script:
    - ${TERRAFORM_BINARY} output -json > outputs.json
```

### tags

**Type:** `string[]`
**Default:** `[]`

GitLab runner tags.

```yaml
gitlab:
  tags:
    - terraform
    - docker
    - aws
```

### variables

**Type:** `map[string]string`
**Default:** `{}`

Additional pipeline variables.

```yaml
gitlab:
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"
    AWS_DEFAULT_REGION: "us-east-1"
```

### artifact_paths

**Type:** `string[]`
**Default:** `["*.tfplan"]`

Artifact paths for plan jobs.

```yaml
gitlab:
  artifact_paths:
    - "*.tfplan"
    - "terraform.tfstate.backup"
```

## Full Example

```yaml
gitlab:
  # Binary configuration
  terraform_binary: "terraform"
  terraform_image: "hashicorp/terraform:1.6"

  # Pipeline structure
  stages_prefix: "deploy"
  parallelism: 5
  plan_enabled: true
  auto_approve: false

  # Scripts
  before_script:
    - export AWS_ROLE_ARN="arn:aws:iam::123456789:role/TerraformRole"
    - ${TERRAFORM_BINARY} init -backend-config="role_arn=${AWS_ROLE_ARN}"

  after_script:
    - echo "Module ${TF_MODULE} completed"

  # Runner configuration
  tags:
    - terraform
    - docker
    - production

  # Variables
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"
    TF_CLI_ARGS_plan: "-parallelism=30"
    TF_CLI_ARGS_apply: "-parallelism=30"

  # Artifacts
  artifact_paths:
    - "*.tfplan"
```

## Generated Output

With the above configuration, TerraCi generates:

```yaml
variables:
  TERRAFORM_BINARY: "terraform"
  TF_IN_AUTOMATION: "true"
  TF_INPUT: "false"
  TF_CLI_ARGS_plan: "-parallelism=30"
  TF_CLI_ARGS_apply: "-parallelism=30"

default:
  image: hashicorp/terraform:1.6
  before_script:
    - export AWS_ROLE_ARN="arn:aws:iam::123456789:role/TerraformRole"
    - ${TERRAFORM_BINARY} init -backend-config="role_arn=${AWS_ROLE_ARN}"
  after_script:
    - echo "Module ${TF_MODULE} completed"
  tags:
    - terraform
    - docker
    - production

stages:
  - deploy-plan-0
  - deploy-apply-0
  # ...

plan-platform-prod-vpc:
  stage: deploy-plan-0
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan
  variables:
    TF_MODULE: vpc
    # ...
  artifacts:
    paths:
      - platform/prod/us-east-1/vpc/*.tfplan
    expire_in: 1 day

apply-platform-prod-vpc:
  stage: deploy-apply-0
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} apply plan.tfplan
  needs:
    - plan-platform-prod-vpc
  when: manual
  # ...
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
