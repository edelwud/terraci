# Policy Configuration

TerraCi integrates [Open Policy Agent (OPA)](https://www.openpolicyagent.org/) to enforce compliance rules on Terraform plans. Policies are written in [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/), OPA's declarative policy language.

## Basic Configuration

```yaml
policy:
  enabled: true
  sources:
    - path: policies
  namespaces:
    - terraform
  on_failure: block
```

## Configuration Options

### enabled

Enable or disable policy checks globally.

```yaml
policy:
  enabled: true  # default: false
```

### sources

List of policy sources. TerraCi supports three source types:

#### Local Path

```yaml
policy:
  sources:
    - path: policies           # Relative to project root
    - path: /absolute/path     # Absolute path
```

#### Git Repository

```yaml
policy:
  sources:
    - git: https://github.com/org/terraform-policies.git
      ref: main                # Branch, tag, or commit SHA
```

#### OCI Registry

```yaml
policy:
  sources:
    - oci: oci://ghcr.io/org/policies:v1.0
```

### namespaces

Rego package namespaces to evaluate. TerraCi looks for `deny` and `warn` rules in these namespaces.

```yaml
policy:
  namespaces:
    - terraform              # Evaluates data.terraform.deny, data.terraform.warn
    - terraform.aws          # Evaluates data.terraform.aws.deny, etc.
    - terraform.security
```

Default: `["terraform"]`

### on_failure

Action when policy check fails (deny rules triggered):

| Value | Description |
|-------|-------------|
| `block` | Fail the pipeline (exit code 1) |
| `warn` | Log warnings but continue (exit code 0) |
| `ignore` | Silently ignore failures |

```yaml
policy:
  on_failure: block  # default
```

### on_warning

Action when policy check has warnings (warn rules triggered):

```yaml
policy:
  on_warning: warn  # default
```

### show_in_comment

Include policy results in MR comment.

```yaml
policy:
  show_in_comment: true  # default: true
```

### cache_dir

Directory for caching downloaded policies.

```yaml
policy:
  cache_dir: .terraci/policies  # default
```

### overwrites

Override policy settings for specific modules using glob patterns:

```yaml
policy:
  enabled: true
  on_failure: block

  overwrites:
    # Allow sandbox deployments with warnings only
    - match: "*/sandbox/*"
      on_failure: warn

    # Skip policy checks for legacy modules
    - match: "legacy/*"
      enabled: false

    # Different namespaces for specific modules
    - match: "platform/prod/*"
      namespaces:
        - terraform
        - terraform.production
```

## Writing Policies

Policies must use OPA v1 Rego syntax with `contains` and `if` keywords.

### Deny Rules

Deny rules block deployment when matched:

```rego
package terraform

import rego.v1

deny contains msg if {
    resource := input.resource_changes[_]
    resource.type == "aws_s3_bucket"
    resource.change.after.acl == "public-read"
    msg := sprintf("S3 bucket '%s' must not be public", [resource.name])
}
```

### Warn Rules

Warn rules generate warnings without blocking:

```rego
package terraform

import rego.v1

warn contains msg if {
    resource := input.resource_changes[_]
    resource.type == "aws_instance"
    not resource.change.after.tags.Environment
    msg := sprintf("Instance '%s' should have Environment tag", [resource.name])
}
```

### Input Structure

Policies receive Terraform plan JSON as input. Key fields:

```json
{
  "format_version": "1.1",
  "resource_changes": [
    {
      "type": "aws_s3_bucket",
      "name": "example",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {
          "bucket": "my-bucket",
          "acl": "private"
        }
      }
    }
  ],
  "planned_values": { ... },
  "configuration": { ... }
}
```

See [Terraform JSON output format](https://developer.hashicorp.com/terraform/internals/json-format) for complete schema.

## Generated Pipeline

When policy checks are enabled, TerraCi adds a `policy-check` stage:

```yaml
stages:
  - deploy-plan-0
  - deploy-plan-1
  - policy-check    # After all plans
  - deploy-apply-0
  - deploy-apply-1
  - summary

policy-check:
  stage: policy-check
  script:
    - terraci policy pull
    - terraci policy check
  needs:
    - job: plan-vpc
      optional: true
    - job: plan-eks
      optional: true
  artifacts:
    paths:
      - .terraci/policy-results.json
    when: always
```

## Commands

### Pull Policies

Download policies from configured sources:

```bash
terraci policy pull
```

### Check Policies

Run policy checks against Terraform plans:

```bash
# Check all modules with plan.json
terraci policy check

# Check specific module
terraci policy check --module platform/prod/eu-central-1/vpc

# Output as JSON
terraci policy check --output json
```

## MR Integration

Policy results are included in the MR comment:

```markdown
### ❌ Policy Check

**3** modules checked: ✅ **1** passed | ⚠️ **1** warned | ❌ **1** failed

<details>
<summary>❌ Failures (1)</summary>

**platform/prod/eu-central-1/vpc:**
- `terraform`: S3 bucket 'logs' must not be public

</details>
```

## Examples

See [examples/policy-checks](https://github.com/edelwud/terraci/tree/main/examples/policy-checks) for:

- Complete `.terraci.yaml` configuration
- Example Rego policies for AWS resources
- GitLab CI pipeline setup
