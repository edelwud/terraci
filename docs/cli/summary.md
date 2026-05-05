---
title: terraci summary
description: Post plan results, cost estimates, and policy checks as MR/PR comments
outline: deep
---

# terraci summary

Posts terraform plan results as a comment on GitLab Merge Requests or GitHub Pull Requests.

## Synopsis

```bash
terraci summary [flags]
```

## Description

The `summary` command collects terraform plan results from artifacts and creates or updates a summary comment on the merge request (GitLab) or pull request (GitHub).

This command is designed to run as a final job in the CI pipeline after all plan jobs have completed. It loads plan results from each module's plan artifacts, enriches them with `{producer}-report.json` files (cost, policy, tfupdate) discovered in the service directory, writes a canonical `summary-report.json`, and posts a formatted MR/PR comment.

The command automatically detects the CI provider and whether it is running in an MR/PR pipeline, and only creates comments when appropriate.

## Usage

This command is typically used in the generated pipeline's summary job.

### GitLab CI

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

### GitHub Actions

```yaml
summary:
  runs-on: ubuntu-latest
  needs: [plan-jobs...]
  if: github.event_name == 'pull_request'
  steps:
    - uses: actions/checkout@v4
    - run: terraci summary
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Environment Variables

### GitLab

| Variable | Description | Required |
|----------|-------------|----------|
| `CI_MERGE_REQUEST_IID` | MR number (auto-detected by GitLab) | Yes |
| `CI_PROJECT_ID` | Project ID (auto-detected) | Yes |
| `CI_SERVER_URL` | GitLab server URL (auto-detected) | Yes |
| `GITLAB_TOKEN` | API token for posting comments | No* |
| `CI_JOB_TOKEN` | Fallback token (auto-provided) | No* |

*Either `GITLAB_TOKEN` or `CI_JOB_TOKEN` is required.

### GitHub

| Variable | Description | Required |
|----------|-------------|----------|
| `GITHUB_ACTIONS` | Indicates GitHub Actions environment (auto-set) | Yes |
| `GITHUB_TOKEN` | Token for posting PR comments | Yes |
| `GITHUB_REPOSITORY` | Repository in `owner/repo` format (auto-set) | Yes |
| `GITHUB_EVENT_PATH` | Path to event payload JSON (auto-set) | Yes |

## Output

The command posts a comment like this to the MR/PR:

```markdown
## 🔄 Terraform Plan Summary

| Module | Status | Summary |
|--------|--------|---------|
| `platform/stage/eu-central-1/vpc` | ✅ Changes | Plan: 2 to add, 1 to change, 0 to destroy |
| `platform/stage/eu-central-1/eks` | ➖ No changes | Infrastructure is up-to-date |

<details>
<summary>📋 platform/stage/eu-central-1/vpc</summary>

Plan: 2 to add, 1 to change, 0 to destroy.
...

</details>
```

## Configuration

The `summary` plugin is **enabled by default** — no explicit `enabled: true` is required. It can be disabled via `extensions.summary`:

```yaml
extensions:
  summary:
    enabled: false  # disable the summary plugin
```

Configure the MR/PR comment behavior via the CI provider config in `.terraci.yaml`:

### GitLab

```yaml
extensions:
  gitlab:
    mr:
      comment:
        enabled: true
        on_changes_only: false
        include_details: true
```

See [GitLab MR Configuration](/config/gitlab-mr) for full options.

### GitHub

```yaml
extensions:
  github:
    pr:
      comment:
        enabled: true
        on_changes_only: false
```

See [GitHub Actions Configuration](/config/github) for full options.

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success (or skipped if not in MR/PR) |
| 1 | Error scanning plan results or posting comment |

## Examples

### Manual Run (for testing)

```bash
# Set required environment variables
export CI_MERGE_REQUEST_IID=42
export CI_PROJECT_ID=12345
export GITLAB_TOKEN=your-token

terraci summary
```

### With Verbose Output

```bash
terraci summary -v
```

## See Also

- [GitLab MR Integration](/config/gitlab-mr)
- [GitHub Actions Configuration](/config/github)
- [terraci generate](/cli/generate)
