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

**Type:** `string` or `object`
**Default:** `"hashicorp/terraform:1.6"`
**Required:** Yes

Docker image for Terraform jobs. Supports both simple string format and object format with entrypoint override.

**String format** (simple):
```yaml
gitlab:
  # Terraform
  terraform_image: "hashicorp/terraform:1.6"

  # OpenTofu
  terraform_image: "ghcr.io/opentofu/opentofu:1.6"

  # Custom image
  terraform_image: "registry.example.com/terraform:1.6"
```

**Object format** (with entrypoint):
```yaml
gitlab:
  # OpenTofu minimal image requires entrypoint override
  terraform_image:
    name: "ghcr.io/opentofu/opentofu:1.9-minimal"
    entrypoint: [""]

  # Custom image with specific entrypoint
  terraform_image:
    name: "registry.example.com/terraform:1.6"
    entrypoint: ["/bin/sh", "-c"]
```

::: tip OpenTofu Minimal Images
OpenTofu minimal images (e.g., `opentofu:1.9-minimal`) have a non-shell entrypoint. Use the object format with `entrypoint: [""]` to override it for GitLab CI compatibility.
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

### id_tokens

**Type:** `map[string]object`
**Default:** `{}`

OIDC tokens for cloud provider authentication. This enables passwordless authentication with AWS, GCP, Azure, and other providers that support OIDC.

```yaml
gitlab:
  id_tokens:
    AWS_OIDC_TOKEN:
      aud: "https://gitlab.example.com"
    GCP_OIDC_TOKEN:
      aud: "https://iam.googleapis.com/projects/123456/locations/global/workloadIdentityPools/gitlab-pool/providers/gitlab"
```

Generated output:

```yaml
default:
  id_tokens:
    AWS_OIDC_TOKEN:
      aud: "https://gitlab.example.com"
```

::: tip AWS OIDC Authentication
Use `id_tokens` with AWS IAM roles for secure, credential-free authentication:
```yaml
gitlab:
  id_tokens:
    AWS_OIDC_TOKEN:
      aud: "https://gitlab.example.com"
  before_script:
    - >
      export $(printf "AWS_ACCESS_KEY_ID=%s AWS_SECRET_ACCESS_KEY=%s AWS_SESSION_TOKEN=%s"
      $(aws sts assume-role-with-web-identity
      --role-arn ${AWS_ROLE_ARN}
      --role-session-name "GitLabRunner-${CI_PROJECT_ID}-${CI_PIPELINE_ID}"
      --web-identity-token ${AWS_OIDC_TOKEN}
      --duration-seconds 3600
      --query 'Credentials.[AccessKeyId,SecretAccessKey,SessionToken]'
      --output text))
    - ${TERRAFORM_BINARY} init
```
:::

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

### secrets

**Type:** `map[string]object`
**Default:** `{}`

Secrets from external secret managers (HashiCorp Vault). Secrets are injected as environment variables or files.

**Shorthand format** (recommended):
```yaml
gitlab:
  secrets:
    credentials:
      vault: ci/terraform/gitlab-terraform/credentials@cdp
      file: true
    API_KEY:
      vault: production/api/keys/main@team
```

**Full format** (for complex configurations):
```yaml
gitlab:
  secrets:
    AWS_SECRET_ACCESS_KEY:
      vault:
        engine:
          name: kv-v2
          path: secret
        path: aws/credentials
        field: secret_access_key
    DATABASE_PASSWORD:
      vault:
        engine:
          name: kv-v2
          path: secret
        path: production/database
        field: password
      file: true  # Write to file instead of env var
```

The shorthand format `path/to/secret/field@namespace` is the standard GitLab syntax.

::: warning Vault Configuration
Secrets require GitLab Vault integration to be configured. See [GitLab Vault documentation](https://docs.gitlab.com/ee/ci/secrets/index.html).
:::

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
