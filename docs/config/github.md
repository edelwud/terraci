---
title: GitHub Actions Configuration
description: "Configure GitHub Actions workflow generation: runners, steps, jobs, overwrites, and permissions"
outline: deep
---

# GitHub Actions Configuration

The `github` section configures the generated GitHub Actions workflow. This section is used when the resolved provider is `github` (set via `provider: github` in config, or auto-detected from the `GITHUB_ACTIONS` environment variable). When the provider is `gitlab`, this section is omitted and the `gitlab` section is used instead. See [GitLab CI Configuration](/config/gitlab) for the GitLab equivalent.

## Options

### terraform_binary

**Type:** `string`
**Default:** `"terraform"`

The Terraform/OpenTofu binary to use.

```yaml
github:
  terraform_binary: "terraform"  # or "tofu"
```

### runs_on

**Type:** `string`
**Default:** `"ubuntu-latest"`

The GitHub Actions runner label for jobs.

```yaml
github:
  runs_on: "ubuntu-latest"
  # runs_on: "self-hosted"
```

### container

**Type:** `object` (optional)
**Default:** none

Optionally run jobs inside a container. Supports both string and object format.

```yaml
github:
  container:
    name: "hashicorp/terraform:1.6"
    entrypoint: [""]
```

### env

**Type:** `map[string]string`
**Default:** `{}`

Workflow-level environment variables.

```yaml
github:
  env:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"
    AWS_DEFAULT_REGION: "us-east-1"
```

### plan_enabled

**Type:** `boolean`
**Default:** `true`

Generate separate plan jobs.

```yaml
github:
  plan_enabled: true   # plan + apply jobs
  # plan_enabled: false  # apply only
```

### plan_only

**Type:** `boolean`
**Default:** `false`

Generate only plan jobs without apply jobs.

```yaml
github:
  plan_only: true
```

### auto_approve

**Type:** `boolean`
**Default:** `false`

Auto-approve apply jobs without environment protection.

```yaml
github:
  auto_approve: false  # Apply uses environment protection
  # auto_approve: true   # Apply runs automatically
```

### init_enabled

**Type:** `boolean`
**Default:** `true`

Automatically run `terraform init` before terraform commands.

```yaml
github:
  init_enabled: true
```

### permissions

**Type:** `map[string]string`
**Default:** `{}`

Workflow-level permissions. Required for PR comments and OIDC authentication.

```yaml
github:
  permissions:
    contents: read
    pull-requests: write
    id-token: write        # Required for OIDC
```

### job_defaults

**Type:** `object`
**Default:** `null`

Default settings applied to all generated jobs (both plan and apply). These are applied before `overwrites`.

Available fields:
- `runs_on` - Override runner label for all jobs
- `container` - Container image for all jobs
- `env` - Additional environment variables
- `steps_before` - Extra steps to run before terraform commands
- `steps_after` - Extra steps to run after terraform commands

**Example: Common setup steps for all jobs**
```yaml
github:
  job_defaults:
    steps_before:
      - uses: actions/checkout@v4
      - uses: hashicorp/setup-terraform@v3
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::123456789012:role/terraform
          aws-region: us-east-1
    steps_after:
      - name: Upload logs
        run: echo "Job completed"
```

Each step in `steps_before` / `steps_after` supports:
- `name` - Step display name
- `uses` - GitHub Action reference (e.g., `actions/checkout@v4`)
- `with` - Action inputs as key-value pairs
- `run` - Shell command to run
- `env` - Step-level environment variables

### overwrites

**Type:** `array`
**Default:** `[]`

Job-level overrides for plan or apply jobs. Applied after `job_defaults`.

Each overwrite has:
- `type` - Which jobs to override: `plan` or `apply`
- `runs_on` - Override runner label
- `container` - Override container image
- `env` - Override/add environment variables
- `steps_before` - Override steps before terraform commands
- `steps_after` - Override steps after terraform commands

**Example: Different runners for plan and apply**
```yaml
github:
  overwrites:
    - type: plan
      runs_on: ubuntu-latest

    - type: apply
      runs_on: self-hosted
      env:
        DEPLOY_ENV: "production"
```

**Example: Extra steps for apply jobs**
```yaml
github:
  overwrites:
    - type: apply
      steps_before:
        - uses: actions/checkout@v4
        - uses: hashicorp/setup-terraform@v3
        - name: Approve deployment
          run: echo "Deploying..."
```

### pr

**Type:** `object`
**Default:** `null`

Pull request integration settings. Equivalent to GitLab's `mr` section.

```yaml
github:
  pr:
    comment:
      enabled: true
      on_changes_only: false
    summary_job:
      runs_on: ubuntu-latest
```

#### pr.comment

Controls PR comment behavior:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | true | Enable PR comments |
| `on_changes_only` | bool | false | Only comment when there are changes |
| `include_details` | bool | true | Include full plan output in expandable sections |

#### pr.summary_job

Configures the summary job that posts PR comments:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `runs_on` | string | `ubuntu-latest` | Runner label for the summary job |

## Full Example

```yaml
provider: github

github:
  # Binary configuration
  terraform_binary: "terraform"
  runs_on: "ubuntu-latest"

  # Workflow settings
  plan_enabled: true
  auto_approve: false
  init_enabled: true

  # Workflow-level environment variables
  env:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"

  # Permissions (required for PR comments and OIDC)
  permissions:
    contents: read
    pull-requests: write
    id-token: write

  # Job defaults (applied to all jobs)
  job_defaults:
    steps_before:
      - uses: actions/checkout@v4
      - uses: hashicorp/setup-terraform@v3
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::123456789012:role/terraform
          aws-region: us-east-1

  # Job overwrites (override job_defaults for specific job types)
  overwrites:
    - type: apply
      runs_on: self-hosted

  # Pull request integration
  pr:
    comment:
      enabled: true
      on_changes_only: false
    summary_job:
      runs_on: ubuntu-latest
```

## Per-Job Environment Variables

Like GitLab, each job receives environment variables dynamically generated from your `structure.pattern` segments. For the default pattern `{service}/{environment}/{region}/{module}`:

| Variable | Description | Example |
|----------|-------------|---------|
| `TF_MODULE_PATH` | Relative path to module | `platform/prod/us-east-1/vpc` |
| `TF_SERVICE` | Service name | `platform` |
| `TF_ENVIRONMENT` | Environment name | `prod` |
| `TF_REGION` | Region name | `us-east-1` |
| `TF_MODULE` | Module name | `vpc` |

Variable names are derived by uppercasing the segment name and prefixing with `TF_`.

## Comparison with GitLab Configuration

| Feature | GitLab (`gitlab:`) | GitHub (`github:`) |
|---------|-------------------|-------------------|
| Runner selection | `job_defaults.tags` | `runs_on` |
| Container image | `image` | `container` (optional) |
| Pre-job commands | `job_defaults.before_script` | `job_defaults.steps_before` |
| Post-job commands | `job_defaults.after_script` | `job_defaults.steps_after` |
| Pipeline variables | `variables` | `env` |
| Access control | `rules` | `permissions` |
| MR/PR integration | `mr` section | `pr` section |
| Secrets | `secrets` (Vault) | Use GitHub Action steps |
| OIDC tokens | `id_tokens` | `permissions.id-token: write` |
| Caching | `cache_enabled` | Use `actions/cache` in steps |
| Stages prefix | `stages_prefix` | N/A (uses job dependencies) |

## See Also

- [GitLab CI Configuration](/config/gitlab) — the equivalent configuration for GitLab CI
- [Merge Request Integration](/config/gitlab-mr) — MR comments with plan summaries and policy results
- [Pipeline Generation Guide](/guide/pipeline-generation) — end-to-end guide for generating CI pipelines
