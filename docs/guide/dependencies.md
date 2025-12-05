# Dependency Resolution

TerraCi automatically discovers dependencies between Terraform modules by analyzing `terraform_remote_state` data sources.

## How It Works

### 1. Parse Remote State References

TerraCi parses all `.tf` files in each module looking for `terraform_remote_state` data sources:

```hcl
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "my-terraform-state"
    key    = "platform/production/us-east-1/vpc/terraform.tfstate"
    region = "us-east-1"
  }
}
```

### 2. Extract State Paths

From each remote state block, TerraCi extracts:
- **Backend type** (s3, gcs, azurerm, etc.)
- **State file path** (from `key`, `prefix`, or similar)
- **Whether `for_each` is used**

### 3. Match to Modules

The state path is matched against discovered modules:

```
key: platform/production/us-east-1/vpc/terraform.tfstate
     ↓
Module ID: platform/production/us-east-1/vpc
```

### 4. Build Dependency Graph

Dependencies are added to a directed acyclic graph (DAG):

```
eks → vpc     (eks depends on vpc)
rds → vpc     (rds depends on vpc)
app → eks    (app depends on eks)
app → rds    (app depends on rds)
```

## Supported Backends

TerraCi extracts state paths from these backends:

| Backend | Path Field |
|---------|------------|
| s3 | `key` |
| gcs | `prefix` |
| azurerm | `key` |
| http | `address` |
| consul | `path` |

## Dynamic References with `for_each`

TerraCi handles `for_each` in remote state blocks:

```hcl
locals {
  dependencies = {
    vpc = "platform/production/us-east-1/vpc"
    iam = "platform/production/us-east-1/iam"
  }
}

data "terraform_remote_state" "deps" {
  for_each = local.dependencies

  backend = "s3"
  config = {
    bucket = "my-terraform-state"
    key    = "${each.value}/terraform.tfstate"
  }
}
```

This creates dependencies on both `vpc` and `iam` modules.

## Locals Resolution

TerraCi resolves local variable references in state paths:

```hcl
locals {
  env        = "production"
  region     = "us-east-1"
  state_key  = "platform/${local.env}/${local.region}/vpc/terraform.tfstate"
}

data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = local.state_key
  }
}
```

## Name-Based Fallback

If the state path can't be matched to a module, TerraCi falls back to name-based matching:

```hcl
# Module: platform/production/us-east-1/eks

data "terraform_remote_state" "vpc" {  # ← name "vpc"
  # ...
}
```

TerraCi looks for a module named `vpc` in the same service/environment/region.

## Submodule Dependencies

For submodules, TerraCi also tries pattern matching:

```hcl
# In module: platform/production/us-east-1/ec2/rabbitmq

data "terraform_remote_state" "ec2_base" {
  # ...
}
```

Matches:
- `ec2_base` → `ec2/base` (submodule pattern)
- `ec2-base` → `ec2/base` (dash-separated)

## Cycle Detection

TerraCi detects circular dependencies:

```bash
terraci validate
```

Output:
```
✗ Circular dependency detected:
  module-a → module-b → module-c → module-a
```

Circular dependencies prevent pipeline generation.

## Visualization

Export the dependency graph to visualize:

```bash
# DOT format for GraphViz
terraci graph --format dot -o deps.dot
dot -Tpng deps.dot -o deps.png

# Simple text format
terraci graph --format list

# Show execution levels
terraci graph --format levels
```

## Execution Levels

TerraCi groups modules into execution levels:

```
Level 0: [vpc, iam]           # No dependencies
Level 1: [eks, rds]           # Depend on level 0
Level 2: [app]                # Depends on level 1
```

Modules at the same level can run in parallel.

## Troubleshooting

### Dependency Not Detected

1. Check that the state path matches a module ID:
   ```bash
   terraci validate -v
   ```

2. Verify the path pattern:
   ```yaml
   backend:
     key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
   ```

3. Check for typos in the remote state config

### Too Many Dependencies

If unintended dependencies are detected:

1. Review the remote state `key` values
2. Ensure state paths match the expected pattern
3. Check for shared state files being referenced

### Missing Module

If a referenced module isn't discovered:

1. Verify the module exists at the correct depth
2. Check that it contains `.tf` files
3. Ensure it's not excluded by filter patterns
