# Structure Configuration

The `structure` section defines how TerraCi discovers Terraform modules.

## Options

### pattern

**Type:** `string`
**Default:** `"{service}/{environment}/{region}/{module}"`

Defines the directory structure pattern using placeholders:

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{service}` | Service or project name | `platform`, `analytics` |
| `{environment}` | Environment name | `production`, `staging` |
| `{region}` | Cloud region | `us-east-1`, `eu-west-1` |
| `{module}` | Module name | `vpc`, `eks`, `rds` |

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
```

### min_depth

**Type:** `integer`
**Default:** Calculated from pattern (4 for default pattern)

Minimum directory depth for module discovery.

```yaml
structure:
  min_depth: 4  # service/env/region/module
```

### max_depth

**Type:** `integer`
**Default:** `min_depth + 1` if `allow_submodules`, else `min_depth`

Maximum directory depth for module discovery. Set to `min_depth + 1` to enable submodules.

```yaml
structure:
  max_depth: 5  # Allows service/env/region/module/submodule
```

### allow_submodules

**Type:** `boolean`
**Default:** `true`

Enable discovery of nested submodules at depth 5.

```yaml
structure:
  allow_submodules: true
```

## Examples

### Standard 4-Level Structure

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 4
  allow_submodules: false
```

```
platform/production/us-east-1/vpc/
platform/production/us-east-1/eks/
```

### With Submodules

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true
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
  min_depth: 3
  max_depth: 3
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
  min_depth: 5
  max_depth: 6
```

```
acme/platform/production/us-east-1/vpc/
acme/platform/production/us-east-1/eks/
```

### Simple Flat Structure

```yaml
structure:
  pattern: "{environment}/{module}"
  min_depth: 2
  max_depth: 2
```

```
production/vpc/
production/eks/
staging/vpc/
```

## Auto-Calculation

If `min_depth` is not specified, it's calculated from the pattern:

| Pattern | Calculated min_depth |
|---------|---------------------|
| `{env}/{module}` | 2 |
| `{env}/{region}/{module}` | 3 |
| `{service}/{env}/{region}/{module}` | 4 |

If `max_depth` is not specified:
- With `allow_submodules: true`: `max_depth = min_depth + 1`
- With `allow_submodules: false`: `max_depth = min_depth`

## Directory Requirements

For a directory to be recognized as a module:

1. **Depth** - Must be between `min_depth` and `max_depth`
2. **Files** - Must contain at least one `.tf` file
3. **Visibility** - Must not be hidden (no `.` prefix)

## Troubleshooting

### Modules Not Found

```bash
terraci validate -v
```

Check:
1. Directory depth matches configuration
2. Directories contain `.tf` files
3. Directories aren't in exclude patterns

### Wrong Module IDs

If module IDs don't match expected paths:

1. Verify the pattern matches your structure
2. Check depth calculations
3. Ensure consistent directory naming
