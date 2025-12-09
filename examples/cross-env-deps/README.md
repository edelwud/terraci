# Cross-Environment Dependencies Example

This example demonstrates how TerraCi handles dependencies that cross environment or region boundaries.

## Structure

```
cdp/
├── stage/eu-central-1/
│   ├── vpc/                    # VPC in stage environment
│   └── ec2/db-migrate/         # Submodule that depends on multiple VPCs
└── vpn/eu-north-1/
    └── vpc/                    # VPC in vpn environment (different region)
```

## The db-migrate Module

The `ec2/db-migrate` submodule references two VPCs:

1. **Same environment VPC** - uses dynamic path with `local.*` variables
2. **VPN VPC** - uses hardcoded cross-environment path

```hcl
# Same environment/region dependency
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = "${local.service}/${local.environment}/${local.region}/vpc/terraform.tfstate"
  }
}

# Cross-environment dependency
data "terraform_remote_state" "vpn_vpc" {
  backend = "s3"
  config = {
    key = "${local.service}/vpn/eu-north-1/vpc/terraform.tfstate"
  }
}
```

## Running the Example

```bash
cd examples/cross-env-deps

# Show dependencies for db-migrate
terraci graph --module cdp/stage/eu-central-1/ec2/db-migrate

# Output:
# Dependencies of cdp/stage/eu-central-1/ec2/db-migrate:
#   - cdp/stage/eu-central-1/vpc
#   - cdp/vpn/eu-north-1/vpc
```

## Key Points

- TerraCi resolves `local.*` variables from the module path structure
- Both dynamic (`${local.service}/...`) and hardcoded paths are supported
- Cross-environment dependencies are properly detected and included in pipelines
