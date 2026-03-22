---
title: Structure Configuration
description: Configure directory patterns for Terraform module discovery
outline: deep
---

# Structure Configuration

The `structure` section defines how TerraCi discovers Terraform modules.

## Options

### pattern

**Type:** `string`
**Default:** `"{service}/{environment}/{region}/{module}"`

Defines the directory structure pattern using placeholders. Segment names are user-defined and any name works -- they are not limited to the defaults below.

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{service}` | Service or project name | `platform`, `analytics` |
| `{environment}` | Environment name | `production`, `staging` |
| `{region}` | Cloud region | `us-east-1`, `eu-west-1` |
| `{module}` | Module name | `vpc`, `eks`, `rds` |

You can use any custom segment names:

```yaml
# Custom segment names
structure:
  pattern: "{team}/{stack}/{datacenter}/{component}"
```

Each segment name in the pattern becomes:
- A per-job environment variable in the generated pipeline (e.g., `TF_TEAM`, `TF_STACK`, `TF_DATACENTER`, `TF_COMPONENT`)
- A filterable key via the `--filter` CLI flag (e.g., `--filter team=infra`)

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
```

## Examples

### Standard 4-Level Structure

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
```

```
platform/production/us-east-1/vpc/
platform/production/us-east-1/eks/
```

### With Submodules

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
```

```
platform/production/us-east-1/ec2/
platform/production/us-east-1/ec2/rabbitmq/
platform/production/us-east-1/ec2/redis/
```

### 3-Level Structure

```yaml
structure:
  pattern: "{environment}/{region}/{module}"
```

```
production/us-east-1/vpc/
production/us-east-1/eks/
staging/us-east-1/vpc/
```

### 5-Level Structure

```yaml
structure:
  pattern: "{org}/{service}/{environment}/{region}/{module}"
```

```
acme/platform/production/us-east-1/vpc/
acme/platform/production/us-east-1/eks/
```

### Simple Flat Structure

```yaml
structure:
  pattern: "{environment}/{module}"
```

```
production/vpc/
production/eks/
staging/vpc/
```

## Directory Requirements

For a directory to be recognized as a module:

1. **Depth** - Must match the number of segments in the pattern (directories with `.tf` files at that depth are modules; deeper directories are submodules)
2. **Files** - Must contain at least one `.tf` file
3. **Visibility** - Must not be hidden (no `.` prefix)

## Troubleshooting

### Modules Not Found

```bash
terraci validate -v
```

Check:
1. Directory depth matches the pattern segment count
2. Directories contain `.tf` files
3. Directories aren't in exclude patterns

### Wrong Module IDs

If module IDs don't match expected paths:

1. Verify the pattern matches your structure
2. Ensure the pattern matches your directory structure
3. Ensure consistent directory naming

## See Also

- [Filters](/config/filters) — include/exclude glob patterns for module filtering
- [Project Structure Guide](/guide/project-structure) — best practices for organizing Terraform modules
