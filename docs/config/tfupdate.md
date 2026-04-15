---
title: Dependency Updates
description: "Terraform dependency resolution and lock synchronization: targets, bump levels, registries, lock file sync, and CI pipeline integration"
outline: deep
---

# Dependency Update Configuration

TerraCi can resolve Terraform provider and module version constraints against the Terraform registry, optionally write updated constraints back to `.tf` files, and synchronize `.terraform.lock.hcl` lock files.

## Basic Configuration

```yaml
plugins:
  tfupdate:
    enabled: true
    policy:
      bump: minor
```

## Configuration Options

### enabled

Enable or disable dependency update checks globally.

```yaml
plugins:
  tfupdate:
    enabled: true  # default: false
```

### target

What dependency types to check.

```yaml
plugins:
  tfupdate:
    target: all  # default: all
```

| Value | Description |
|-------|-------------|
| `all` | Check both providers and modules (default) |
| `providers` | Check only `required_providers` blocks |
| `modules` | Check only `module` source version references |

### policy.bump

The version bump level that determines which updates are proposed.

```yaml
plugins:
  tfupdate:
    policy:
      bump: minor  # required
```

| Value | Description |
|-------|-------------|
| `patch` | Propose patch-level updates only |
| `minor` | Propose minor and patch updates |
| `major` | Propose major, minor, and patch updates |

### policy.pin

Pin updated dependency constraints to an exact version when writing.

```yaml
plugins:
  tfupdate:
    policy:
      bump: minor
      pin: false  # default: false
```

When `true`, constraints like `~> 5.80` are replaced with `5.80.0` on write.

### ignore

List of provider sources or module sources to skip during checks. Useful for internal registries or pinned dependencies that should not be updated.

```yaml
plugins:
  tfupdate:
    ignore:
      - registry.terraform.io/hashicorp/null
      - github.com/internal/terraform-aws-vpc
```

Each entry is matched against the full source string of the provider or module.

### timeout

Overall timeout for a tfupdate run. Defaults to 5 minutes in read-only mode and 20 minutes in write mode.

```yaml
plugins:
  tfupdate:
    timeout: "15m"
```

### registries

Configure custom registry hostnames for provider lookups.

```yaml
plugins:
  tfupdate:
    registries:
      default: registry.terraform.io  # default
      providers:
        hashicorp/aws: custom-registry.example.com
```

| Field | Description |
|-------|-------------|
| `default` | Default registry hostname for modules/providers without lock-based host information |
| `providers` | Per-provider registry hostname overrides keyed by short source (e.g., `hashicorp/aws`) |

### lock

Configure lock file synchronization behavior.

```yaml
plugins:
  tfupdate:
    lock:
      platforms:
        - linux_amd64
        - darwin_arm64
```

| Field | Description |
|-------|-------------|
| `platforms` | Platform set for provider h1 hashes in `.terraform.lock.hcl`. Empty means all available platforms. |

### cache

Configure caching for registry metadata and provider archives.

```yaml
plugins:
  tfupdate:
    cache:
      metadata:
        backend: inmemcache     # default: inmemcache
        ttl: "6h"               # default: 6h
        namespace: tfupdate/registry
      artifacts:
        backend: diskblob       # default: diskblob
        namespace: tfupdate/providers
```

| Field | Default | Description |
|-------|---------|-------------|
| `cache.metadata.backend` | `inmemcache` | KV cache backend plugin name for registry metadata |
| `cache.metadata.ttl` | `6h` | How long registry metadata stays cached |
| `cache.metadata.namespace` | `tfupdate/registry` | Namespace for metadata cache entries |
| `cache.artifacts.backend` | `diskblob` | Blob store backend for downloaded provider archives |
| `cache.artifacts.namespace` | `tfupdate/providers` | Namespace for cached provider archives and hashes |

### pipeline

Add a dependency update check job to the generated CI pipeline.

```yaml
plugins:
  tfupdate:
    pipeline: false  # default: false
```

When `true`, TerraCi adds a `tfupdate-check` job to the pipeline that runs `terraci tfupdate` in read-only mode and saves results as a CI artifact.

## Full Example

```yaml
plugins:
  tfupdate:
    enabled: true
    target: all
    policy:
      bump: minor
      pin: false
    ignore:
      - registry.terraform.io/hashicorp/null
      - registry.terraform.io/hashicorp/random
    registries:
      default: registry.terraform.io
    lock:
      platforms:
        - linux_amd64
        - darwin_arm64
    cache:
      metadata:
        backend: inmemcache
        ttl: "6h"
      artifacts:
        backend: diskblob
    pipeline: false
    timeout: "15m"
```

## CLI Usage

The tfupdate plugin exposes the `terraci tfupdate` command:

```bash
# Check all providers and modules
terraci tfupdate

# Check providers only, patch-level
terraci tfupdate --target providers --bump patch

# Apply minor updates in-place and sync lock files
terraci tfupdate --write

# Pin constraints to exact versions
terraci tfupdate --write --pin

# Check a specific module
terraci tfupdate --module platform/prod/eu-central-1/vpc

# Specify platforms for lock file hashing
terraci tfupdate --lock-platforms linux_amd64,darwin_arm64

# JSON output
terraci tfupdate --output json
```

See [terraci tfupdate](/cli/tfupdate) for full CLI reference.

## Version Constraint Handling

TerraCi recognizes all standard Terraform version constraint operators: `~>`, `>=`, `<=`, `>`, `<`, `=`, `!=`. Comma-separated constraints such as `">= 1.0, < 2.0"` are also supported.

When `--write` is applied, constraint style is preserved — only the version value is updated, keeping the original operator and format intact. For example, `~> 5.0` bumped to `5.82` becomes `~> 5.82`, and `>= 1.0` bumped to `2.0` becomes `>= 2.0`.

## Lock File Synchronization

When `--write` is used, TerraCi automatically updates `.terraform.lock.hcl` files alongside `.tf` constraint changes:

1. For each updated provider, the lock file entry is created or updated with the new version.
2. `zh:` hashes are collected from registry metadata for all available platforms.
3. `h1:` hashes are computed by downloading provider archives for the configured `lock.platforms` (or all platforms if not configured).
4. Existing hashes in the lock file are preserved and merged with new ones.

This ensures that `terraform init` will not fail due to stale lock file entries after an update.

## How It Works

1. Modules are discovered using the configured `structure.pattern` and filter rules.
2. For each module, TerraCi reads `required_providers` blocks, `module` source references, and `.terraform.lock.hcl`.
3. The planner/solver resolves compatible version selections considering transitive provider constraints from module dependencies.
4. Provider and module versions are resolved via the Terraform registry.
5. Current constraints are compared to the latest available version matching the `bump` level.
6. Results are output to the terminal and saved as artifacts in the service directory.
7. In write mode, `.tf` files and `.terraform.lock.hcl` are updated atomically.

Registry lookups are parallelized and cached per run to minimize network round-trips.

## Artifacts

After each run, two files are written to the service directory (`.terraci/` by default):

| File | Description |
|------|-------------|
| `tfupdate-results.json` | Full structured results for all checked dependencies |
| `tfupdate-report.json` | Summary report for CI comment integration |

## See Also

- [terraci tfupdate](/cli/tfupdate) — CLI reference for the tfupdate command
- [Configuration Overview](/config/) — full configuration reference
