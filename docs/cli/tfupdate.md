---
title: terraci tfupdate
description: Resolve Terraform provider and module versions, optionally writing version bumps and lock file updates back to .tf files
outline: deep
---

# terraci tfupdate

Resolve Terraform provider and module versions against the Terraform registry, with optional in-place updates and lock file synchronization.

## Synopsis

```bash
terraci tfupdate [flags]
```

## Description

The `tfupdate` command scans all discovered modules for Terraform provider and module version constraints, queries the Terraform registry for the latest available versions, and reports what can be updated.

Default mode is read-only and reports available updates without modifying any files. Use `--write` to apply version bumps in-place to matching `.tf` files and synchronize `.terraform.lock.hcl` lock files.

Exit behavior:
- Exits `0` when the scan completes without operational errors
- Exits non-zero when parse, registry, or write errors are encountered
- Available updates alone do not cause the command to fail

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--target` | `-t` | string | `all` | What to check: `modules`, `providers`, `all` |
| `--bump` | `-b` | string | | Version bump level: `patch`, `minor`, `major` |
| `--write` | `-w` | bool | false | Write updated versions back to `.tf` files and sync lock files |
| `--pin` | | bool | false | Pin updated constraints to an exact version when writing |
| `--module` | `-m` | string | | Check a specific module path only |
| `--output` | `-o` | string | `text` | Output format: `text`, `json` |
| `--timeout` | | string | | Overall timeout for the run (e.g., `15m`) |
| `--lock-platforms` | | []string | | Platforms for lock file h1 hashes (e.g., `linux_amd64,darwin_arm64`) |

## Examples

```bash
# Check all providers and modules for updates
terraci tfupdate

# Check only providers
terraci tfupdate --target providers

# Check only modules
terraci tfupdate --target modules

# Check only patch-level updates
terraci tfupdate --bump patch

# Apply updates in-place to .tf files and sync lock files
terraci tfupdate --write

# Pin constraints to exact versions
terraci tfupdate --write --pin

# Check and write minor updates for providers only
terraci tfupdate --target providers --bump minor --write

# Specify platforms for lock file hashing
terraci tfupdate --write --lock-platforms linux_amd64,darwin_arm64

# Set a custom timeout
terraci tfupdate --timeout 15m

# Check a single module
terraci tfupdate --module platform/prod/eu-central-1/vpc

# JSON output (for scripts/CI)
terraci tfupdate --output json
```

## Output

### Text (default)

```
• updates available   modules=2
  • platform/prod/eu-central-1/vpc   updates=2
    • hashicorp/aws registry.terraform.io/hashicorp/aws   current=~> 5.0   available=~> 5.80
    • vpc github.com/terraform-aws-modules/terraform-aws-vpc   current=~> 5.0   available=~> 5.19
  • platform/prod/eu-central-1/eks   updates=1
    • hashicorp/aws registry.terraform.io/hashicorp/aws   current=~> 5.0   available=~> 5.80
• summary
  • checked   count=47
  • updates available   count=3
```

When all dependencies are up to date:

```
• summary
  • checked   count=47
• all dependencies are up to date
```

When `--write` is applied, each updated entry includes a `status=applied` field.

### JSON

```bash
terraci tfupdate --output json
```

```json
{
  "providers": [
    {
      "module_path": "platform/prod/eu-central-1/vpc",
      "provider_name": "aws",
      "source": "registry.terraform.io/hashicorp/aws",
      "current_constraint": "~> 5.0",
      "available_constraint": "~> 5.80",
      "status": "available"
    }
  ],
  "modules": [
    {
      "module_path": "platform/prod/eu-central-1/vpc",
      "call_name": "vpc",
      "source": "github.com/terraform-aws-modules/terraform-aws-vpc",
      "current_constraint": "~> 5.0",
      "available_constraint": "~> 5.19",
      "status": "available"
    }
  ],
  "summary": {
    "total_checked": 47,
    "updates_available": 3,
    "updates_applied": 0,
    "errors": 0,
    "skipped": 0
  }
}
```

## Version Bump Levels

The `--bump` flag controls the maximum version constraint change proposed:

| Level | Example: current `~> 5.0` | Result |
|-------|--------------------------|--------|
| `patch` | `5.80.3` available | `~> 5.80` |
| `minor` | `5.80.0` available | `~> 5.80` (default) |
| `major` | `6.0.0` available | `~> 6.0` |

## Version Constraint Handling

TerraCi recognizes all standard Terraform version constraint operators: `~>`, `>=`, `<=`, `>`, `<`, `=`, `!=`. Comma-separated constraints (e.g., `">= 1.0, < 2.0"`) are also supported.

When a constraint like `~> 5.80` is encountered, the base version is extracted as `5.80` for registry comparison.

### Constraint Style Preservation

When `--write` is used, the original constraint style is preserved:

| Current | Latest available | Result |
|---------|-----------------|--------|
| `~> 5.0` | `5.82.0` | `~> 5.82` (minor, keeps `~> X.Y` format) |
| `~> 5.0.1` | `5.1.3` | `~> 5.1.3` (patch-level, keeps 3-part format) |
| `>= 1.0` | `2.0.0` | `>= 2.0` (preserves operator) |

Only the version value changes; the constraint operator and format are kept as-is.

## Write Mode

When `--write` is passed, TerraCi updates version constraints directly in `.tf` files:

```hcl
# Before
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# After (with --write --bump minor)
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.80"
    }
  }
}
```

Only the version constraint value is modified; all other content is preserved.

## Lock File Synchronization

When `--write` is used, `.terraform.lock.hcl` files are automatically updated alongside `.tf` constraint changes:

- Provider lock entries are created or updated with the new version
- `zh:` hashes are collected from registry metadata for all available platforms
- `h1:` hashes are computed by downloading provider archives for the configured platforms
- Existing hashes are preserved and merged with new ones

Use `--lock-platforms` to restrict which platforms get `h1:` hashes (download-heavy). Platforms not in the list still get `zh:` hashes from registry metadata.

## Prerequisites

- `extensions.tfupdate.enabled: true` in `.terraci.yaml`
- `extensions.tfupdate.policy.bump` must be set (via config or `--bump` flag)
- Network access to the Terraform registry (`registry.terraform.io`) for provider lookups
- Network access to the appropriate registry for module lookups

## Artifacts

After each run, TerraCi writes two files to the service directory (`.terraci/` by default):

- `tfupdate-results.json` — full results for further processing
- `tfupdate-report.json` — summary report for CI integration

## See Also

- [Tfupdate configuration](/config/tfupdate) — enable and configure the tfupdate plugin
