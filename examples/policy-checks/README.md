# Policy Checks Example

This example demonstrates how to use TerraCi's OPA-based policy checks to enforce compliance rules on your Terraform plans.

## Overview

TerraCi integrates [Open Policy Agent (OPA)](https://www.openpolicyagent.org/) to evaluate Rego policies against Terraform plan JSON output. This allows you to:

- Block deployments that violate security policies
- Warn about potential issues without blocking
- Enforce organizational standards across all modules

## Files

- `.terraci.yaml` - TerraCi configuration with policy settings
- `.gitlab-ci.yml` - GitLab CI pipeline with policy-check stage
- `policies/` - Directory containing Rego policy files

## Policy Configuration

```yaml
policy:
  enabled: true
  sources:
    - path: policies  # Local policies
  namespaces:
    - terraform       # Namespace to evaluate
  on_failure: block   # Block pipeline on violations
  on_warning: warn    # Continue with warnings
  show_in_comment: true
```

### Policy Sources

TerraCi supports multiple policy sources:

```yaml
policy:
  sources:
    # Local path
    - path: policies

    # Git repository
    - git: https://github.com/org/terraform-policies.git
      ref: main

    # OCI registry
    - oci: oci://ghcr.io/org/policies:v1.0
```

### Module-Level Overwrites

You can override policy settings for specific modules using glob patterns:

```yaml
policy:
  enabled: true
  on_failure: block

  overwrites:
    - match: "*/sandbox/*"
      on_failure: warn  # Allow sandbox deployments with warnings

    - match: "legacy/*"
      enabled: false    # Skip policy checks for legacy modules
```

## Writing Policies

Policies are written in [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/), OPA's policy language.

### Deny Rules (Block Deployment)

```rego
package terraform

# Block public S3 buckets
deny contains msg if {
    resource := input.resource_changes[_]
    resource.type == "aws_s3_bucket"
    resource.change.after.acl == "public-read"
    msg := sprintf("S3 bucket '%s' must not be public", [resource.name])
}
```

### Warn Rules (Allow with Warning)

```rego
package terraform

# Warn about missing tags
warn contains msg if {
    resource := input.resource_changes[_]
    resource.type == "aws_instance"
    not resource.change.after.tags.Environment
    msg := sprintf("Instance '%s' is missing Environment tag", [resource.name])
}
```

## Pipeline Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Terraform Pipeline                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  plan-0: [plan-vpc, plan-rds, ...]                          │
│                         │                                    │
│                         ▼                                    │
│  policy-check:                                               │
│    - terraci policy pull    (download policies)             │
│    - terraci policy check   (evaluate all plans)            │
│                         │                                    │
│                         ▼                                    │
│  apply-0: [apply-vpc, apply-rds, ...]  (if policy passes)   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Commands

```bash
# Pull policies from configured sources
terraci policy pull

# Check all modules with plan.json
terraci policy check

# Check specific module
terraci policy check --module platform/prod/eu-central-1/vpc

# Output as JSON
terraci policy check --output json
```

## MR Integration

When running in a GitLab MR, policy results are included in the MR comment:

```
## Terraform Plan Summary

### ❌ Policy Check

**3** modules checked: ✅ **1** passed | ⚠️ **1** warned | ❌ **1** failed

<details>
<summary>❌ Failures (1)</summary>

**platform/prod/eu-central-1/vpc:**
- `terraform`: S3 bucket 'logs' must not be public

</details>

<details>
<summary>⚠️ Warnings (1)</summary>

**platform/prod/eu-central-1/ec2:**
- `terraform`: Instance 'web' is missing Environment tag

</details>
```

## Example Policies

See the `policies/` directory for example Rego policies:

- `s3.rego` - S3 bucket security rules
- `ec2.rego` - EC2 instance compliance rules
- `tags.rego` - Required tagging policies
