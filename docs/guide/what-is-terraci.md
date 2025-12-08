# What is TerraCi?

TerraCi is a CLI tool that analyzes Terraform/OpenTofu projects and automatically generates GitLab CI pipelines with proper dependency ordering.

## The Problem

When managing infrastructure as code in a monorepo with multiple Terraform modules, you face several challenges:

1. **Dependency Management** - Modules often depend on each other (e.g., EKS depends on VPC). Running them in the wrong order causes failures.

2. **Manual Pipeline Maintenance** - Writing and maintaining GitLab CI pipelines manually is error-prone and tedious.

3. **Change Detection** - When only one module changes, you don't want to re-apply all modules.

4. **Parallel Execution** - Independent modules should run in parallel to reduce deployment time.

## The Solution

TerraCi solves these problems by:

### 1. Automatic Module Discovery

TerraCi scans your directory structure to find all Terraform modules:

```
infrastructure/
├── platform/
│   ├── stage/
│   │   └── eu-central-1/
│   │       ├── vpc/        ← Module discovered
│   │       ├── eks/        ← Module discovered
│   │       └── rds/        ← Module discovered
```

### 2. Dependency Extraction

It parses `terraform_remote_state` data sources to understand which modules depend on which:

```hcl
# In eks/main.tf
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = "platform/stage/eu-central-1/vpc/terraform.tfstate"
  }
}
```

TerraCi detects that `eks` depends on `vpc`.

### 3. Topological Sorting

Using Kahn's algorithm, TerraCi sorts modules into execution levels:

```
Level 0: vpc (no dependencies)
Level 1: eks, rds (depend on vpc)
Level 2: app (depends on eks and rds)
```

### 4. Pipeline Generation

Finally, it generates a GitLab CI pipeline where:
- Modules at the same level run in parallel
- Modules wait for their dependencies to complete
- Plan and apply stages are separated (optional)

## Key Features

| Feature | Description |
|---------|-------------|
| **Smart Discovery** | Finds modules at depth 4 and 5 (with submodules) |
| **Dependency Graph** | Builds accurate DAG from remote state references |
| **Cycle Detection** | Warns about circular dependencies |
| **Git Integration** | Detects changed modules from git diff |
| **OpenTofu Support** | Works with both Terraform and OpenTofu |
| **Glob Filtering** | Include/exclude modules with patterns |
| **DOT Export** | Visualize dependencies with GraphViz |

## When to Use TerraCi

TerraCi is ideal for:

- **Monorepos** with multiple Terraform modules
- **Teams** that need consistent CI/CD pipelines
- **Complex infrastructures** with many interdependencies
- **GitLab CI** users (GitHub Actions support planned)

## Requirements

- Go 1.22+ (for building from source)
- GitLab CI (for pipeline execution)
- Terraform or OpenTofu modules using `terraform_remote_state`
