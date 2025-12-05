# terraci validate

Validate project structure and dependencies.

## Synopsis

```bash
terraci validate [flags]
```

## Description

The `validate` command checks your project for:
- Module discovery correctness
- Dependency graph validity
- Circular dependency detection
- Configuration errors

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--verbose` | `-v` | bool | false | Show detailed output |

## Examples

### Basic Validation

```bash
terraci validate
```

Output:
```
✓ Found 12 modules
✓ Built dependency graph with 15 edges
✓ No circular dependencies detected
✓ 4 execution levels identified

Validation passed
```

### Verbose Output

```bash
terraci validate -v
```

Output:
```
Configuration:
  Pattern: {service}/{environment}/{region}/{module}
  Min depth: 4
  Max depth: 5

Discovered modules:
  - platform/production/us-east-1/vpc
  - platform/production/us-east-1/eks
  - platform/production/us-east-1/rds
  - platform/production/us-east-1/app
  ...

Dependency graph:
  eks → vpc
  rds → vpc
  app → eks
  app → rds

Execution levels:
  Level 0: [vpc]
  Level 1: [eks, rds]
  Level 2: [app]

✓ Validation passed
```

### With Circular Dependencies

```bash
terraci validate
```

Output:
```
✓ Found 5 modules
✓ Built dependency graph with 6 edges
✗ Circular dependency detected:
  module-a → module-b → module-c → module-a

Validation failed
```

## What Gets Validated

### 1. Configuration

- Pattern is valid
- Depth values are correct
- Required fields are present

### 2. Module Discovery

- Modules exist at expected depths
- Modules contain .tf files
- Module IDs are unique

### 3. Dependency Graph

- Remote state references resolve to modules
- No circular dependencies exist
- Execution levels can be calculated

### 4. Execution Order

- Topological sort succeeds
- All modules can be ordered

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Validation passed |
| 1 | General error |
| 2 | Configuration error |
| 3 | Validation error (cycles, etc.) |

## Use Cases

### Pre-Commit Hook

```bash
#!/bin/sh
terraci validate || exit 1
```

### CI Validation

```yaml
validate:
  stage: test
  script:
    - terraci validate -v
```

### Debug Module Discovery

```bash
terraci validate -v 2>&1 | grep "Discovered modules" -A 100
```

## Troubleshooting

### No Modules Found

```
✗ Found 0 modules
```

Check:
1. Directory structure matches pattern
2. Modules contain .tf files
3. Depth configuration is correct

### Unresolved Dependencies

```
Warning: Remote state 'vpc' in module 'eks' could not be resolved
```

Check:
1. State file key matches module path
2. Module exists at expected path
3. Pattern configuration is correct

### Circular Dependencies

```
✗ Circular dependency detected
```

Review the cycle path and fix remote_state references to break the cycle.
