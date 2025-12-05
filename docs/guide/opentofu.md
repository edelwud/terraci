# OpenTofu Support

TerraCi has first-class support for [OpenTofu](https://opentofu.org/), the open-source Terraform fork.

## Configuration

Switch to OpenTofu by updating `.terraci.yaml`:

```yaml
gitlab:
  terraform_binary: "tofu"
  terraform_image: "ghcr.io/opentofu/opentofu:1.6"
```

## How It Works

When you set `terraform_binary: "tofu"`, TerraCi:

1. Sets `TERRAFORM_BINARY=tofu` in the pipeline variables
2. Uses `${TERRAFORM_BINARY}` in all generated scripts
3. Generates commands like `tofu init`, `tofu plan`, `tofu apply`

## Generated Pipeline

```yaml
variables:
  TERRAFORM_BINARY: "tofu"

default:
  image: ghcr.io/opentofu/opentofu:1.6
  before_script:
    - ${TERRAFORM_BINARY} init

plan-platform-prod-vpc:
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan
  # ...

apply-platform-prod-vpc:
  script:
    - cd platform/prod/us-east-1/vpc
    - ${TERRAFORM_BINARY} apply plan.tfplan
  # ...
```

## Official OpenTofu Images

Use official OpenTofu Docker images:

| Image | Description |
|-------|-------------|
| `ghcr.io/opentofu/opentofu:latest` | Latest stable |
| `ghcr.io/opentofu/opentofu:1.6` | Version 1.6.x |
| `ghcr.io/opentofu/opentofu:1.6.0` | Specific version |

## Mixed Environments

If you have both Terraform and OpenTofu modules, you can override per-job:

```yaml
# In your custom pipeline template
.tofu-job:
  variables:
    TERRAFORM_BINARY: "tofu"
  image: ghcr.io/opentofu/opentofu:1.6

.terraform-job:
  variables:
    TERRAFORM_BINARY: "terraform"
  image: hashicorp/terraform:1.6
```

Then extend the generated jobs as needed.

## State Compatibility

OpenTofu is compatible with Terraform state files. You can:

1. Keep existing Terraform state files
2. Migrate to OpenTofu without state changes
3. Use the same S3/GCS backends

TerraCi's dependency resolution works identically for both.

## Migration Guide

### From Terraform to OpenTofu

1. Update `.terraci.yaml`:
   ```yaml
   gitlab:
     terraform_binary: "tofu"
     terraform_image: "ghcr.io/opentofu/opentofu:1.6"
   ```

2. Regenerate pipelines:
   ```bash
   terraci generate -o .gitlab-ci.yml
   ```

3. Test with dry-run:
   ```bash
   terraci generate --dry-run
   ```

4. Commit and push

### Gradual Migration

Migrate module by module using job overrides:

```yaml
# Override specific jobs to use OpenTofu
apply-platform-prod-vpc:
  extends: .tofu-job
```

## Version Compatibility

TerraCi works with:

| Tool | Supported Versions |
|------|-------------------|
| Terraform | 0.12+ |
| OpenTofu | 1.0+ |

The HCL parsing is compatible with both.

## Custom Binary Path

If your binary has a custom name or path:

```yaml
gitlab:
  terraform_binary: "/usr/local/bin/tofu-1.6"
  before_script:
    - ${TERRAFORM_BINARY} init
```

## Environment Variables

Set OpenTofu-specific environment variables:

```yaml
gitlab:
  variables:
    TERRAFORM_BINARY: "tofu"
    TF_CLI_CONFIG_FILE: "/etc/tofu/config.tfrc"
    TOFU_LOG: "INFO"
```
