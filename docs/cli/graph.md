---
title: terraci graph
description: Visualize module dependency graph in DOT, list, or levels format
outline: deep
---

# terraci graph

Display and export the dependency graph.

## Synopsis

```bash
terraci graph [flags]
```

## Description

The `graph` command visualizes module dependencies in various formats.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--format` | `-F` | string | dot | Output format: `dot`, `plantuml`, `list`, `levels` |
| `--output` | `-o` | string | stdout | Output file path |
| `--stats` | | bool | false | Show graph statistics |
| `--module` | `-m` | string | | Query specific module |
| `--dependents` | | bool | false | Show reverse dependencies |
| `--exclude` | `-x` | string[] | | Exclude patterns |
| `--include` | `-i` | string[] | | Include patterns |
| `--filter` | `-f` | string[] | | Filter by segment (`key=value`) |

## Formats

### DOT Format

GraphViz DOT format for visualization:

```bash
terraci graph --format dot -o deps.dot
terraci graph -F dot -o deps.dot
```

Output:
```txt
digraph dependencies {
  rankdir=LR;
  node [shape=box];

  "platform/prod/us-east-1/vpc";
  "platform/prod/us-east-1/eks";
  "platform/prod/us-east-1/rds";

  "platform/prod/us-east-1/eks" -> "platform/prod/us-east-1/vpc";
  "platform/prod/us-east-1/rds" -> "platform/prod/us-east-1/vpc";
}
```

Render with GraphViz:

```bash
terraci graph -f dot -o deps.dot
dot -Tpng deps.dot -o deps.png
dot -Tsvg deps.dot -o deps.svg
```

### PlantUML Format

PlantUML format for diagram generation:

```bash
terraci graph --format plantuml -o deps.puml
```

Output:
```
@startuml
...
"platform/prod/us-east-1/eks" --> "platform/prod/us-east-1/vpc"
"platform/prod/us-east-1/rds" --> "platform/prod/us-east-1/vpc"
@enduml
```

Render with PlantUML:

```bash
terraci graph -F plantuml -o deps.puml
plantuml deps.puml
```

### List Format

Simple text list:

```bash
terraci graph --format list
```

Output:
```
Dependencies:
  platform/prod/us-east-1/eks → platform/prod/us-east-1/vpc
  platform/prod/us-east-1/rds → platform/prod/us-east-1/vpc
  platform/prod/us-east-1/app → platform/prod/us-east-1/eks
  platform/prod/us-east-1/app → platform/prod/us-east-1/rds
```

### Levels Format

Execution levels:

```bash
terraci graph --format levels
```

Output:
```
Execution Levels:
  Level 0: vpc, iam
  Level 1: eks, rds, cache
  Level 2: app-backend, app-frontend
  Level 3: monitoring
```

## Statistics

```bash
terraci graph --stats
```

Output:
```
dependency graph statistics
  total modules                               count: 12
  total edges                                 count: 15
  root modules (no dependencies)              count: 2
  leaf modules (no dependents)                count: 3
  max depth (execution levels)                depth: 4
  average depth                               depth: 1.8
  no cycles ✓
```

## Module Queries

### Dependencies of a Module

```bash
terraci graph --module platform/prod/us-east-1/app
```

Output:
```
Module: platform/prod/us-east-1/app

Dependencies (this module depends on):
  - platform/prod/us-east-1/eks
  - platform/prod/us-east-1/rds
```

### Dependents of a Module

```bash
terraci graph --module platform/prod/us-east-1/vpc --dependents
```

Output:
```
Module: platform/prod/us-east-1/vpc

Dependents (modules that depend on this):
  - platform/prod/us-east-1/eks
  - platform/prod/us-east-1/rds
  - platform/prod/us-east-1/cache
```

## Examples

### Generate PNG Visualization

```bash
terraci graph -f dot | dot -Tpng > deps.png
```

### Save to File

```bash
terraci graph --format dot --output dependencies.dot
```

### Integration with Other Tools

```bash
# Count dependencies
terraci graph --format list | wc -l

# Find modules with many deps
terraci graph --stats | grep "most dependencies" -A 5

# Filter specific service
terraci graph --format list | grep "platform/"
```

### CI Documentation

```yaml
generate-docs:
  script:
    - terraci graph -f dot -o deps.dot
    - dot -Tsvg deps.dot -o public/dependencies.svg
  artifacts:
    paths:
      - public/dependencies.svg
```

## DOT Customization

The DOT output can be customized with GraphViz:

```bash
# Horizontal layout
terraci graph -f dot | sed 's/rankdir=LR/rankdir=TB/' | dot -Tpng > deps.png

# Different node shapes
terraci graph -f dot | sed 's/shape=box/shape=ellipse/' | dot -Tpng > deps.png

# Colored output
dot -Tpng -Gcolor=lightblue deps.dot -o deps.png
```

## Use Cases

### Documentation

Generate dependency documentation:

```bash
echo "# Dependencies" > DEPENDENCIES.md
echo "" >> DEPENDENCIES.md
echo "![Dependency Graph](deps.svg)" >> DEPENDENCIES.md
terraci graph -f dot | dot -Tsvg > deps.svg
```

### Impact Analysis

See what's affected by changing a module:

```bash
terraci graph -m platform/prod/vpc --dependents
```

### Debugging

Find why a module is included:

```bash
terraci graph -m platform/prod/app
# Shows what app depends on
```

## See Also

- [Dependency Resolution Guide](/guide/dependencies) — how TerraCi resolves module dependencies
