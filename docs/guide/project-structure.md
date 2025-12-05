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

Each module is identified by its path components:

| Path | Service | Environment | Region | Module |
|------|---------|-------------|--------|--------|
| `platform/production/us-east-1/vpc` | platform | production | us-east-1 | vpc |
| `analytics/production/eu-west-1/redshift` | analytics | production | eu-west-1 | redshift |

The module ID is the full path: `platform/production/us-east-1/vpc`

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

  # Minimum depth (auto-calculated from pattern if not set)
  min_depth: 4

  # Maximum depth (for submodule support)
  max_depth: 5

  # Enable submodule discovery
  allow_submodules: true
```

### Custom Patterns

You can customize the pattern for different layouts:

**3-level structure:**
```yaml
structure:
  pattern: "{environment}/{region}/{module}"
  min_depth: 3
  max_depth: 3
```

**5-level structure:**
```yaml
structure:
  pattern: "{org}/{service}/{environment}/{region}/{module}"
  min_depth: 5
  max_depth: 6
```

## What Makes a Module?

A directory is considered a Terraform module if:

1. It's at the correct depth (between `min_depth` and `max_depth`)
2. It contains at least one `.tf` file

TerraCi ignores:
- Hidden directories (starting with `.`)
- Directories without `.tf` files
- Directories outside the depth range

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
  min_depth: 2
  max_depth: 2
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
1. Directory depth matches `min_depth`/`max_depth`
2. Directories contain `.tf` files
3. Directories aren't hidden (no `.` prefix)

### Wrong Module IDs

If module IDs don't match your state file paths, adjust the pattern:

```yaml
backend:
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

This pattern is used to match `terraform_remote_state` keys to modules.
