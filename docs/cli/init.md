# terraci init

Initialize a TerraCi configuration file.

## Synopsis

```bash
terraci init [flags]
```

## Description

The `init` command creates a `.terraci.yaml` configuration file with sensible defaults.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | false | Overwrite existing configuration |

## Examples

### Basic Initialization

```bash
terraci init
```

Creates `.terraci.yaml`:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

exclude:
  - "*/test/*"
  - "*/sandbox/*"

gitlab:
  terraform_binary: "terraform"
  terraform_image: "hashicorp/terraform:1.6"
  stages_prefix: "deploy"
  parallelism: 5
  plan_enabled: true
  auto_approve: false
  before_script:
    - ${TERRAFORM_BINARY} init
  tags:
    - terraform
    - docker
  variables:
    TF_IN_AUTOMATION: "true"
    TF_INPUT: "false"
  artifact_paths:
    - "*.tfplan"

backend:
  type: s3
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

### Force Overwrite

```bash
terraci init --force
```

Overwrites existing `.terraci.yaml` without prompting.

### Different Directory

```bash
terraci -d /path/to/project init
```

Creates configuration in specified directory.

## What Gets Created

The command creates:
- `.terraci.yaml` in the current (or specified) directory

It does NOT modify:
- Existing Terraform files
- GitLab CI configuration
- Any other project files

## After Initialization

1. **Review the configuration**
   ```bash
   cat .terraci.yaml
   ```

2. **Customize for your project**
   - Adjust the pattern to match your structure
   - Update the Docker image
   - Add exclude patterns
   - Configure backend settings

3. **Validate**
   ```bash
   terraci validate
   ```

4. **Generate your first pipeline**
   ```bash
   terraci generate --dry-run
   terraci generate -o .gitlab-ci.yml
   ```

## Configuration Templates

### OpenTofu Setup

After init, modify for OpenTofu:

```yaml
gitlab:
  terraform_binary: "tofu"
  terraform_image: "ghcr.io/opentofu/opentofu:1.6"
```

### GCS Backend

```yaml
backend:
  type: gcs
  bucket: my-terraform-state
  key_pattern: "{service}/{environment}/{region}/{module}/terraform.tfstate"
```

### Custom Structure

```yaml
structure:
  pattern: "{environment}/{region}/{module}"
  min_depth: 3
  max_depth: 3
  allow_submodules: false
```

## Troubleshooting

### File Already Exists

```
Error: .terraci.yaml already exists. Use --force to overwrite.
```

Solution: Use `--force` or manually edit the existing file.

### Permission Denied

```
Error: permission denied: .terraci.yaml
```

Solution: Check file permissions and ownership.
