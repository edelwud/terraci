# terraci summary

Posts terraform plan results as a comment on GitLab Merge Requests.

## Synopsis

```bash
terraci summary [flags]
```

## Description

The `summary` command collects terraform plan results from artifacts and creates or updates a summary comment on the GitLab merge request.

This command is designed to run as a final job in the GitLab CI pipeline after all plan jobs have completed. It scans for `plan.txt` files in module directories and posts a formatted comment to the MR.

The command automatically detects if it's running in a GitLab MR pipeline and only creates comments when appropriate.

## Usage

This command is typically used in the generated pipeline's summary job:

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

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `CI_MERGE_REQUEST_IID` | MR number (auto-detected by GitLab) | Yes |
| `CI_PROJECT_ID` | Project ID (auto-detected) | Yes |
| `CI_SERVER_URL` | GitLab server URL (auto-detected) | Yes |
| `GITLAB_TOKEN` | API token for posting comments | No* |
| `CI_JOB_TOKEN` | Fallback token (auto-provided) | No* |

*Either `GITLAB_TOKEN` or `CI_JOB_TOKEN` is required.

## Output

The command posts a comment like this to the MR:

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

Configure the summary job via `.terraci.yaml`:

```yaml
gitlab:
  mr:
    comment:
      enabled: true
      on_changes_only: false
      include_details: true
    summary_job:
      image:
        name: "ghcr.io/edelwud/terraci:latest"
      tags:
        - docker
```

See [GitLab MR Configuration](/config/gitlab-mr) for full options.

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success (or skipped if not in MR) |
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
- [terraci generate](/cli/generate)
