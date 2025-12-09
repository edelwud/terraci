# Filters Configuration

Filter modules using include and exclude patterns.

## Options

### exclude

**Type:** `string[]`
**Default:** `[]`

Glob patterns for modules to exclude.

```yaml
exclude:
  - "*/test/*"
  - "*/sandbox/*"
  - "*/.terraform/*"
```

### include

**Type:** `string[]`
**Default:** `[]` (all modules)

Glob patterns for modules to include. If empty, all modules are included (after excludes).

```yaml
include:
  - "platform/*/*/*"
  - "analytics/*/*/*"
```

## Pattern Syntax

TerraCi uses glob patterns with these special characters:

| Pattern | Matches |
|---------|---------|
| `*` | Any characters except `/` |
| `**` | Any characters including `/` (any depth) |
| `?` | Single character |
| `[abc]` | Character class |
| `[!abc]` | Negated character class |

## Examples

### Exclude Test Modules

```yaml
exclude:
  - "*/test/*"
  - "*/tests/*"
  - "*-test/*/*/*"
```

### Exclude Specific Environment

```yaml
exclude:
  - "*/sandbox/*"
  - "*/development/*"
```

### Exclude Specific Region

```yaml
exclude:
  - "*/*/eu-north-1/*"
```

### Only Production

```yaml
include:
  - "*/production/*/*"
```

### Only Specific Service

```yaml
include:
  - "platform/*/*/*"
```

### Exclude Submodules

```yaml
exclude:
  - "*/*/*/*/*"  # Exclude depth 5
```

### Only Submodules

```yaml
include:
  - "*/*/*/*/*"  # Include only depth 5
```

## CLI Overrides

Override filters from command line:

```bash
# Add excludes
terraci generate --exclude "*/test/*" --exclude "*/sandbox/*"

# Add includes
terraci generate --include "platform/*/*/*"

# Filter by field
terraci generate --service platform
terraci generate --environment production
terraci generate --region us-east-1
```

## Filter Order

Filters are applied in this order:

1. **Discovery** - Find all modules at correct depth
2. **Exclude** - Remove modules matching exclude patterns
3. **Include** - If set, keep only modules matching include patterns
4. **CLI Filters** - Apply service/environment/region filters

## Field-Based Filters

Filter by module fields via CLI:

```bash
# By service
terraci generate --service platform

# By environment
terraci generate --environment production

# By region
terraci generate --region us-east-1

# Combined
terraci generate --service platform --environment production --region us-east-1
```

## Combining Filters

Filters can be combined:

```yaml
exclude:
  - "*/sandbox/*"
  - "*/test/*"

include:
  - "platform/*/*/*"
  - "analytics/*/*/*"
```

This:
1. Excludes all sandbox and test modules
2. Then includes only platform and analytics modules

## Use Cases

### Production Pipeline

```yaml
exclude:
  - "*/sandbox/*"
  - "*/test/*"
  - "*/development/*"

include:
  - "*/production/*/*"
```

### Regional Deployment

Generate pipeline for specific region:

```bash
terraci generate --region us-east-1
```

Or in config:

```yaml
include:
  - "*/*/us-east-1/*"
```

### Service-Specific Pipeline

```bash
terraci generate --service platform
```

### Exclude Specific Modules

```yaml
exclude:
  - "platform/production/us-east-1/legacy-vpc"
  - "analytics/*/*/deprecated-*"
```

## Debugging Filters

See which modules are included:

```bash
terraci validate -v
```

Output:
```
Discovered 20 modules
After exclude patterns: 15 modules
After include patterns: 10 modules
Final module count: 10

Modules:
  - platform/production/us-east-1/vpc
  - platform/production/us-east-1/eks
  ...
```

## Wildcards in Paths

### Single Level (`*`)

```yaml
include:
  - "platform/*/us-east-1/*"  # Any environment, only us-east-1
```

Matches:
- `platform/production/us-east-1/vpc`
- `platform/staging/us-east-1/vpc`

### Multi-Level (`**`)

```yaml
include:
  - "platform/**"  # All platform modules at any depth
```

Matches:
- `platform/production/us-east-1/vpc`
- `platform/production/us-east-1/ec2/rabbitmq`

## Library Modules

Library modules (also called shared modules) are reusable Terraform modules that don't have their own providers or remote state - they are used by executable modules via the `module` block.

### Configuration

```yaml
library_modules:
  paths:
    - "_modules"
    - "shared/modules"
```

### How It Works

When you configure `library_modules.paths`, TerraCi:

1. **Parses module blocks** in executable modules to find local module calls (`source = "../_modules/kafka"`)
2. **Tracks library dependencies** in the dependency graph
3. **Detects library changes** when using `--changed-only` mode
4. **Includes affected modules** when a library module is modified

### Example Structure

```
terraform/
├── _modules/               # Library modules
│   ├── kafka/              # Reusable Kafka configuration
│   │   └── main.tf
│   └── kafka_acl/          # Kafka ACL module (depends on kafka)
│       └── main.tf
├── platform/
│   └── production/
│       └── eu-north-1/
│           └── msk/        # Executable module using _modules/kafka
│               └── main.tf
```

In `platform/production/eu-north-1/msk/main.tf`:

```hcl
module "kafka" {
  source = "../../../../_modules/kafka"
  # ...
}

module "kafka_acl" {
  source = "../../../../_modules/kafka_acl"
  # ...
}
```

### Change Detection

When you modify `_modules/kafka/main.tf`:

```bash
terraci generate --changed-only
```

TerraCi will include `platform/production/eu-north-1/msk` in the pipeline because it uses the `kafka` library module.

### Transitive Dependencies

If `kafka_acl` library module uses `kafka` module internally, and you modify `kafka`, all modules using `kafka_acl` will also be detected as affected.

### Verbose Output

Use verbose mode to see library module detection:

```bash
terraci generate --changed-only -v
```

Output:
```
Changed library modules: 1
  - /project/_modules/kafka
Affected modules (including dependents): 3
  - platform/production/eu-north-1/msk
  - platform/production/eu-north-1/streaming
  - platform/production/eu-west-1/msk
```

### Example

See the [library-modules example](https://github.com/edelwud/terraci/tree/main/examples/library-modules) for a complete working example.
