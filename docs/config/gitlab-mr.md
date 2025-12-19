# GitLab MR Integration

TerraCi can automatically post terraform plan summaries as comments on GitLab Merge Requests.

## Overview

When running in a GitLab MR pipeline, TerraCi:
1. Captures terraform plan output from each module
2. Collects all results in a summary job
3. Posts a formatted comment to the MR with all plan summaries

## Configuration

### Basic Setup

```yaml
gitlab:
  mr:
    comment:
      enabled: true
    summary_job:
      image:
        name: "ghcr.io/edelwud/terraci:latest"
```

### Full Options

```yaml
gitlab:
  mr:
    # Comment configuration
    comment:
      # Enable MR comments (default: true when mr section exists)
      enabled: true
      # Only comment when there are changes (default: false)
      on_changes_only: false
      # Include full plan output in expandable sections (default: true)
      include_details: true

    # Labels to add to MR (supports placeholders)
    labels:
      - "terraform"
      - "env:{environment}"
      - "service:{service}"

    # Summary job configuration
    summary_job:
      # Docker image containing terraci binary
      image:
        name: "ghcr.io/edelwud/terraci:latest"
      # Runner tags
      tags:
        - docker
```

## Label Placeholders

Labels support the following placeholders that are expanded for each module:

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{service}` | Service name | `platform` |
| `{environment}` | Environment | `production` |
| `{env}` | Short for environment | `prod` |
| `{region}` | Cloud region | `eu-central-1` |
| `{module}` | Module name | `vpc` |

## How It Works

### 1. Plan Jobs

When MR integration is enabled, plan jobs are modified to:
- Use `-detailed-exitcode` flag to detect changes
- Capture output to `plan.txt` file
- Save `plan.txt` as artifact (with `when: always`)

```yaml
plan-platform-stage-eu-central-1-vpc:
  script:
    - cd platform/stage/eu-central-1/vpc
    - ${TERRAFORM_BINARY} init
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan -detailed-exitcode 2>&1 | tee plan.txt; exit ${PIPESTATUS[0]}
  artifacts:
    paths:
      - platform/stage/eu-central-1/vpc/plan.tfplan
      - platform/stage/eu-central-1/vpc/plan.txt
    expire_in: 1 day
    when: always
```

### 2. Summary Job

A `terraci-summary` job is added that:
- Runs after all plan jobs complete
- Only runs in MR pipelines (`$CI_MERGE_REQUEST_IID`)
- Scans for `plan.txt` files from artifacts
- Posts/updates MR comment via GitLab API

```yaml
terraci-summary:
  stage: summary
  image: ghcr.io/edelwud/terraci:latest
  script:
    - terraci summary
  needs:
    - job: plan-platform-stage-eu-central-1-vpc
      optional: true
  rules:
    - if: $CI_MERGE_REQUEST_IID
      when: always
```

### 3. Comment Format

The MR comment includes:
- Overview table with status icons
- Count of modules with changes/no changes/errors
- Expandable details with full plan output (if enabled)

Example:
```
## 🔄 Terraform Plan Summary

| Module | Status | Summary |
|--------|--------|---------|
| `platform/stage/eu-central-1/vpc` | ✅ Changes | Plan: 2 to add, 1 to change, 0 to destroy |
| `platform/stage/eu-central-1/eks` | ➖ No changes | Infrastructure is up-to-date |

<details>
<summary>📋 platform/stage/eu-central-1/vpc</summary>

```
Plan: 2 to add, 1 to change, 0 to destroy.
...
```

</details>
```

## Authentication

The summary job requires a GitLab API token:

### Using CI_JOB_TOKEN

The default `CI_JOB_TOKEN` works for same-project MRs:
```yaml
# No additional configuration needed
```

### Using GITLAB_TOKEN

For cross-project or enhanced permissions:
```yaml
variables:
  GITLAB_TOKEN: $GITLAB_API_TOKEN
```

Required scopes: `api` or `write_repository`

## Environment Variables

The summary job uses these CI/CD variables:

| Variable | Description |
|----------|-------------|
| `CI_MERGE_REQUEST_IID` | MR number (auto-detected) |
| `CI_PROJECT_ID` | Project ID (auto-detected) |
| `CI_PROJECT_PATH` | Project path (auto-detected) |
| `GITLAB_TOKEN` | API token (falls back to `CI_JOB_TOKEN`) |

## Troubleshooting

### Comment Not Posted

1. Check MR integration is enabled:
   ```yaml
   gitlab:
     mr:
       comment:
         enabled: true
   ```

2. Verify running in MR pipeline:
   ```bash
   echo $CI_MERGE_REQUEST_IID
   ```

3. Check token permissions:
   ```bash
   curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
     "https://gitlab.com/api/v4/projects/$CI_PROJECT_ID/merge_requests/$CI_MERGE_REQUEST_IID"
   ```

### No Plan Results Found

1. Verify plan.txt files exist in artifacts:
   ```bash
   find . -name "plan.txt"
   ```

2. Check plan jobs completed (even with failures):
   ```yaml
   artifacts:
     when: always  # Required for failed plans
   ```

### Summary Job Missing

The summary job only appears when:
1. MR integration is enabled (`gitlab.mr` section exists)
2. Plans are enabled (`gitlab.plan_enabled: true`)
