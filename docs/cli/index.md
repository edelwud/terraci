# CLI Reference

TerraCi command-line interface reference.

## Global Options

These options are available for all commands:

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to configuration file |
| `--dir` | `-d` | Working directory |
| `--verbose` | `-v` | Enable verbose output |
| `--help` | `-h` | Show help |

## Commands

| Command | Description |
|---------|-------------|
| [generate](./generate) | Generate GitLab CI pipeline |
| [validate](./validate) | Validate project structure |
| [graph](./graph) | Show dependency graph |
| [init](./init) | Initialize configuration |
| [summary](./summary) | Post plan results to MR |
| `version` | Show version information |

## Usage

```bash
terraci [command] [flags]
```

## Examples

```bash
# Generate pipeline
terraci generate -o .gitlab-ci.yml

# Validate with verbose output
terraci validate -v

# Use custom config
terraci -c custom.yaml generate

# Work in different directory
terraci -d /path/to/project validate
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Validation error (circular dependencies, etc.) |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `TERRACI_CONFIG` | Default config file path |
| `TERRACI_DIR` | Default working directory |

## Version

```bash
terraci version
```

Output:
```
terraci version v0.1.0
  commit: abc1234
  built:  2024-01-15T10:30:00Z
```
