# Library Modules Example

This example demonstrates how TerraCi handles library (shared) modules - reusable Terraform modules that don't have their own providers or state files.

## Structure

```
library-modules/
├── _modules/                           # Library modules directory
│   ├── kafka/                          # Reusable Kafka configuration module
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── outputs.tf
│   └── kafka_acl/                      # Kafka ACL module (uses kafka module)
│       ├── main.tf
│       ├── variables.tf
│       └── outputs.tf
├── platform/
│   └── prod/
│       ├── eu-north-1/
│       │   ├── vpc/                    # VPC module
│       │   └── msk/                    # MSK module using _modules/kafka
│       └── eu-west-1/
│           └── msk/                    # Another MSK module using _modules/kafka
└── .terraci.yaml
```

## Library Modules

Library modules are characterized by:
- No `provider` blocks (providers are configured by calling modules)
- No `terraform_remote_state` references
- Reusable across multiple executable modules

### kafka module

A reusable module for Kafka/MSK configuration:

```hcl
# _modules/kafka/main.tf
variable "cluster_name" {}
variable "vpc_id" {}

resource "aws_msk_configuration" "this" {
  name           = var.cluster_name
  kafka_versions = ["3.5.1"]
  # ...
}
```

### kafka_acl module

A module for Kafka ACLs that depends on the kafka module:

```hcl
# _modules/kafka_acl/main.tf
module "kafka" {
  source = "../kafka"
  # ...
}
```

## Executable Modules

Executable modules use library modules via the `module` block:

```hcl
# platform/prod/eu-north-1/msk/main.tf
module "kafka" {
  source = "../../../../_modules/kafka"

  cluster_name = "prod-eu-north-1"
  vpc_id       = data.terraform_remote_state.vpc.outputs.vpc_id
}

module "kafka_acl" {
  source = "../../../../_modules/kafka_acl"
  # ...
}
```

## Configuration

```yaml
# .terraci.yaml
library_modules:
  paths:
    - "_modules"
```

Listing a directory under `library_modules.paths` is enough — TerraCi automatically excludes everything inside it from execution targets, so there is no need for an `exclude: ["_modules/*"]` workaround. Library modules are still discovered and available to `terraci validate` and `terraci graph` for diagnostics; they just never become root modules in the pipeline.

## Running the Example

```bash
cd examples/library-modules

# Validate the configuration — surfaces a library-modules summary and warns
# about any orphan libraries (no executable consumers).
terraci validate

# Show the dependency graph with library modules rendered as a separate
# dashed cluster and dashed edges to their consumers.
terraci graph --format dot | dot -Tpng -o deps.png

# Generate pipeline (dry run) — library modules are not executable jobs.
terraci generate --dry-run
```

## Change Detection Flow

When you modify `_modules/kafka/main.tf`:

1. TerraCi detects the change in a library module path
2. Finds all executable modules that use `_modules/kafka`:
   - `platform/prod/eu-north-1/msk`
   - `platform/prod/eu-west-1/msk`
3. Also finds modules using `_modules/kafka_acl` (since kafka_acl depends on kafka)
4. Includes all affected modules in the pipeline

## Key Points

- Library modules are tracked separately from executable modules
- Changes to library modules trigger pipelines for all dependent executable modules
- Transitive library dependencies are supported (kafka_acl → kafka)
- Library module paths are configured in `.terraci.yaml`
