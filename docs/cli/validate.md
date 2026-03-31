---
title: terraci validate
description: Validate project structure, configuration, and dependency graph
outline: deep
---

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
| `--exclude` | `-x` | string[] | | Exclude patterns |
| `--include` | `-i` | string[] | | Include patterns |
| `--filter` | `-f` | string[] | | Filter by segment (`key=value`) |

## Examples

### Basic Validation

```bash
terraci validate
```

Output:
```
validating terraform project structure
  dependency links found                      count: 5

validating dependency graph
  no circular dependencies
  root modules (no deps)                      count: 2
  leaf modules (no dependents)                count: 1
  max dependency depth                        depth: 3

checking execution order
  execution levels determined                 levels: 3

validation PASSED
```

### With Circular Dependencies

```bash
terraci validate
```

Output:
```
validating terraform project structure
  dependency links found                      count: 6

validating dependency graph
  circular dependency detected                cycle: module-a → module-b → module-c → module-a

validation FAILED
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
validating terraform project structure
  dependency links found                      count: 0
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
  circular dependency detected                cycle: module-a → module-b → module-a
```

Review the cycle path and fix remote_state references to break the cycle.

## See Also

- [Project Structure Guide](/guide/project-structure) — best practices for organizing Terraform modules
- [Configuration Overview](/config/) — full configuration reference for .terraci.yaml
