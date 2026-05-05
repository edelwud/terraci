---
title: Project Structure
description: "Directory patterns and module discovery for Terraform monorepos"
outline: deep
---

# Project Structure

TerraCi discovers Terraform modules based on your directory structure. This page explains the supported layouts and how to configure them.

## Default Pattern

The default pattern is:

```
{service}/{environment}/{region}/{module}
```

This translates to a 4-level directory structure:

```
infrastructure/
├── platform/              # service
│   ├── production/        # environment
│   │   └── us-east-1/     # region
│   │       ├── vpc/       # module (depth 4)
│   │       ├── eks/       # module (depth 4)
│   │       └── rds/       # module (depth 4)
│   └── staging/
│       └── us-east-1/
│           └── vpc/
└── analytics/
    └── production/
        └── eu-west-1/
            └── redshift/
```

## Module Identification

Each module is identified by its relative path, which also serves as its ID. The path segments are mapped to named components based on the configured pattern:

| Path | `Get("service")` | `Get("environment")` | `Get("region")` | `Get("module")` |
|------|---------|-------------|--------|--------|
| `platform/production/us-east-1/vpc` | platform | production | us-east-1 | vpc |
| `analytics/production/eu-west-1/redshift` | analytics | production | eu-west-1 | redshift |

The module ID is its relative path: `platform/production/us-east-1/vpc`

The segment names are fully configurable. With a pattern like `{team}/{env}/{module}`, the components would be `team`, `env`, and `module` instead.

## Submodules (Depth 5)

TerraCi supports nested submodules at depth 5:

```
infrastructure/
└── platform/
    └── production/
        └── us-east-1/
            └── ec2/                    # parent module (depth 4)
                ├── main.tf             # parent module files
                ├── rabbitmq/           # submodule (depth 5)
                │   └── main.tf
                └── redis/              # submodule (depth 5)
                    └── main.tf
```

In this case:
- `platform/production/us-east-1/ec2` is a base module
- `platform/production/us-east-1/ec2/rabbitmq` is a submodule
- `platform/production/us-east-1/ec2/redis` is a submodule

::: tip
Both the parent module and submodules can exist simultaneously. TerraCi discovers all directories containing `.tf` files within the depth range.
:::

## Configuration

Configure the structure in `.terraci.yaml`:

```yaml
structure:
  # Directory pattern
  pattern: "{service}/{environment}/{region}/{module}"
```

### Custom Patterns

You can customize the pattern for different layouts:

**3-level structure:**
```yaml
structure:
  pattern: "{environment}/{region}/{module}"
```

**5-level structure:**
```yaml
structure:
  pattern: "{org}/{service}/{environment}/{region}/{module}"
```

## What Makes a Module?

A directory is considered a Terraform module if:

1. It's at the depth defined by the pattern (number of segments), or deeper (submodules)
2. It contains at least one `.tf` file

TerraCi ignores:
- Hidden directories (starting with `.`)
- Directories without `.tf` files

## Examples

### Multi-Cloud Setup

```
infrastructure/
├── aws/
│   └── production/
│       └── us-east-1/
│           └── vpc/
└── gcp/
    └── production/
        └── us-central1/
            └── vpc/
```

### Team-Based Structure

```yaml
structure:
  pattern: "{team}/{environment}/{region}/{module}"
```

```
infrastructure/
├── platform/
│   └── prod/
│       └── eu-west-1/
│           └── eks/
└── data/
    └── prod/
        └── eu-west-1/
            └── redshift/
```

### Simple Flat Structure

```yaml
structure:
  pattern: "{environment}/{module}"
```

```
infrastructure/
├── production/
│   ├── vpc/
│   └── eks/
└── staging/
    └── vpc/
```

## Troubleshooting

### Modules Not Discovered

Run validation to see what TerraCi finds:

```bash
terraci validate -v
```

Check:
1. Directory depth matches the pattern
2. Directories contain `.tf` files
3. Directories aren't hidden (no `.` prefix)

### Wrong Module IDs

If module IDs don't match your state file paths, adjust `structure.pattern` to mirror how your modules lay out on disk:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
```

The same pattern is used to match `terraform_remote_state` keys to modules — derive your state keys from the filesystem path (e.g. `abspath(path.module)`) so they line up with the discovered module IDs. See [Dependency Resolution](/guide/dependencies) for the full mechanism.

## Next Steps

- [Dependency Resolution](/guide/dependencies) — Learn how TerraCi extracts and resolves module dependencies
- [Filters Configuration](/config/filters) — Include or exclude modules with glob patterns
