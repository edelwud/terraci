---
title: Dependency Updates
description: "Terraform provider and module version checking: targets, bump levels, ignore lists, and CI pipeline integration"
outline: deep
---

# Dependency Update Configuration

TerraCi can check Terraform provider and module version constraints against the Terraform registry and optionally write updated constraints back to `.tf` files.

## Basic Configuration

```yaml
plugins:
  update:
    enabled: true
```

## Configuration Options

### enabled

Enable or disable dependency update checks globally.

```yaml
plugins:
  update:
    enabled: true  # default: false
```

### target

What dependency types to check.

```yaml
plugins:
  update:
    target: all  # default: all
```

| Value | Description |
|-------|-------------|
| `all` | Check both providers and modules (default) |
| `providers` | Check only `required_providers` blocks |
| `modules` | Check only `module` source version references |

### bump

The version bump level that determines which updates are proposed.

```yaml
plugins:
  update:
    bump: minor  # default: minor
```

| Value | Description |
|-------|-------------|
| `patch` | Propose patch-level updates only |
| `minor` | Propose minor and patch updates (default) |
| `major` | Propose major, minor, and patch updates |

Minor is the default to avoid unexpected breaking changes from major version upgrades.

### ignore

List of provider sources or module sources to skip during checks. Useful for internal registries or pinned dependencies that should not be updated.

```yaml
plugins:
  update:
    ignore:
      - registry.terraform.io/hashicorp/null
      - github.com/internal/terraform-aws-vpc
```

Each entry is matched against the full source string of the provider or module.

### pipeline

Add a dependency update check job to the generated CI pipeline.

```yaml
plugins:
  update:
    pipeline: false  # default: false
```

When `true`, TerraCi adds an `update-check` job to the pipeline that runs `terraci update` in read-only mode and saves results as a CI artifact.

## Full Example

```yaml
plugins:
  update:
    enabled: true
    target: all
    bump: minor
    ignore:
      - registry.terraform.io/hashicorp/null
      - registry.terraform.io/hashicorp/random
    pipeline: false
```

## CLI Usage

The update plugin exposes the `terraci update` command:

```bash
# Check all providers and modules
terraci update

# Check providers only, patch-level
terraci update --target providers --bump patch

# Apply minor updates in-place
terraci update --write

# Check a specific module
terraci update --module platform/prod/eu-central-1/vpc

# JSON output
terraci update --output json
```

See [terraci update](/cli/update) for full CLI reference.

## Version Constraint Handling

TerraCi recognizes all standard Terraform version constraint operators: `~>`, `>=`, `<=`, `>`, `<`, `=`, `!=`. Comma-separated constraints such as `">= 1.0, < 2.0"` are also supported.

When `--write` is applied, constraint style is preserved — only the version value is updated, keeping the original operator and format intact. For example, `~> 5.0` bumped to `5.82` becomes `~> 5.82`, and `>= 1.0` bumped to `2.0` becomes `>= 2.0`.

## How It Works

1. Modules are discovered using the configured `structure.pattern` and filter rules.
2. For each module, TerraCi reads `required_providers` blocks and `module` source references.
3. Provider versions are resolved via the Terraform registry (`registry.terraform.io`).
4. Module versions are resolved from the respective source registry.
5. Current constraints are compared to the latest available version matching the `bump` level.
6. Results are output to the terminal and saved as artifacts in the service directory.

Registry lookups are parallelized and cached per run to minimize network round-trips.

## Artifacts

After each run, two files are written to the service directory (`.terraci/` by default):

| File | Description |
|------|-------------|
| `update-results.json` | Full structured results for all checked dependencies |
| `update-report.json` | Summary report for CI comment integration |

## See Also

- [terraci update](/cli/update) — CLI reference for the update command
- [Configuration Overview](/config/) — full configuration reference
