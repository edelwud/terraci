---
title: terraci init
description: Initialize .terraci.yaml configuration via interactive TUI wizard or CLI flags
outline: deep
---

# terraci init

Initialize a TerraCi configuration file.

## Synopsis

```bash
terraci init [flags]
```

## Description

The `init` command creates a `.terraci.yaml` configuration file. By default, it launches an interactive TUI wizard that guides you through configuration choices. Use `--ci` for non-interactive mode suitable for automation, or pass specific flags to skip the wizard.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--force` | `-f` | bool | false | Overwrite existing configuration |
| `--ci` | | bool | false | Non-interactive mode (skip TUI wizard) |
| `--provider` | | string | | CI provider: `gitlab` or `github` |
| `--binary` | | string | | Terraform binary: `terraform` or `tofu` |
| `--image` | | string | | Docker image for CI jobs |
| `--pattern` | | string | | Directory structure pattern |

When any of `--provider`, `--binary`, `--image`, or `--pattern` is provided, the wizard is automatically skipped and non-interactive mode is used.

## Examples

### Interactive Mode (Default)

```bash
terraci init
```

This launches a TUI wizard that prompts you to select:
1. CI provider (GitLab CI or GitHub Actions)
2. Terraform binary (Terraform or OpenTofu)
3. Directory structure pattern
4. Whether to enable MR/PR comments
6. Whether to enable cost estimation

### Non-Interactive Mode

```bash
terraci init --ci
```

Creates `.terraci.yaml` with default values without prompting.

### Provider Selection

```bash
# Generate config for GitHub Actions
terraci init --provider github

# Generate config for GitLab CI
terraci init --provider gitlab
```

When `--provider github` is used, the generated config will have a `github:` section (with `runs_on`, `steps_before`, etc.) and no `gitlab:` section. When `--provider gitlab` is used, the config will have a `gitlab:` section (with `image`, `tags`, etc.) and no `github:` section.

### OpenTofu Setup

```bash
terraci init --provider gitlab --binary tofu
```

This automatically selects the appropriate image (`ghcr.io/opentofu/opentofu:1.6`) and for GitHub Actions, uses `opentofu/setup-opentofu@v1` instead of `hashicorp/setup-terraform@v3`.

### Custom Image

```bash
terraci init --binary terraform --image registry.example.com/terraform:1.6
```

### Custom Pattern

```bash
terraci init --pattern "{team}/{stack}/{datacenter}/{component}"
```

### Full Non-Interactive Example

```bash
terraci init --provider github --binary tofu --pattern "{service}/{environment}/{region}/{module}"
```

### Force Overwrite

```bash
terraci init --force
```

Overwrites existing `.terraci.yaml` without prompting.

### Different Directory

```bash
terraci -d /path/to/project init
```

Creates configuration in specified directory.

## Generated Configuration

### GitLab Provider

When `--provider gitlab` (or default), creates:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

extensions:
  gitlab:
    terraform_binary: "terraform"
    image: "hashicorp/terraform:1.6"
    plan_enabled: true
    auto_approve: false
    init_enabled: true
    mr:
      comment:
        enabled: true

backend:
  type: s3
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

### GitHub Provider

When `--provider github`, creates:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"

extensions:
  github:
    terraform_binary: "terraform"
    runs_on: "ubuntu-latest"
    plan_enabled: true
    auto_approve: false
    init_enabled: true
    permissions:
      contents: read
      pull-requests: write
    job_defaults:
      steps_before:
        - uses: actions/checkout@v4
        - uses: hashicorp/setup-terraform@v3
    pr:
      comment: {}

backend:
  type: s3
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

## What Gets Created

The command creates:
- `.terraci.yaml` in the current (or specified) directory

It does NOT modify:
- Existing Terraform files
- CI configuration files
- Any other project files

## After Initialization

1. **Review the configuration**
   ```bash
   cat .terraci.yaml
   ```

2. **Customize for your project**
   - Adjust the pattern to match your structure
   - Update the Docker image or runner labels
   - Add exclude patterns
   - Configure backend settings

3. **Validate**
   ```bash
   terraci validate
   ```

4. **Generate your first pipeline**
   ```bash
   terraci generate --dry-run
   # For GitLab:
   terraci generate -o .gitlab-ci.yml
   # For GitHub:
   terraci generate -o .github/workflows/terraform.yml
   ```

## Troubleshooting

### File Already Exists

```
Error: config file already exists: .terraci.yaml (use --force to overwrite)
```

Solution: Use `--force` or manually edit the existing file.

### Permission Denied

```
Error: permission denied: .terraci.yaml
```

Solution: Check file permissions and ownership.

## See Also

- [Configuration Overview](/config/) -- full configuration reference for .terraci.yaml
- [Getting Started](/guide/getting-started) -- step-by-step guide to set up TerraCi
