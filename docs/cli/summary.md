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

This command is designed to run as a resource-dependent DAG job after the plan and report artifacts it consumes are available. It loads plan results from each module's plan artifacts, enriches them with `{producer}-report.json` files (cost, policy, tfupdate) discovered in the service directory, and posts a formatted MR/PR comment.

The command automatically detects the CI provider and whether it is running in an MR/PR pipeline, and only creates comments when appropriate.

## Usage

This command is typically used in the generated pipeline's summary job.

### GitLab CI

```yaml
terraci-summary:
  stage: deploy-3
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

Configure MR/PR comment behavior through the `summary` plugin in `.terraci.yaml`:

```yaml
extensions:
  summary:
    enabled: true
    on_changes_only: false
    include_details: true
```

GitLab/GitHub providers only supply the comment transport and CI context; they do not own summary rendering options.

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

- [Summary Configuration](/config/summary)
- [GitHub Actions Configuration](/config/github)
- [terraci generate](/cli/generate)
