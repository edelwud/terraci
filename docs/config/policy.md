---
title: Policy Checks
description: "OPA policy configuration: sources, namespaces, enforcement actions, overrides, and MR/PR integration"
outline: deep
---

# Policy Configuration

TerraCi integrates [Open Policy Agent (OPA)](https://www.openpolicyagent.org/) to enforce compliance rules on Terraform plans. Policies are written in [Rego v1](https://www.openpolicyagent.org/docs/latest/policy-language/) syntax.

## Basic Configuration

```yaml
extensions:
  policy:
    enabled: true
    sources:
      - type: path
        path: terraform           # directory name = Rego package name
    namespaces:
      - terraform
    decisions:
      deny: block
      warn: warn
```

## Configuration Options

### enabled

Enable or disable policy checks globally.

```yaml
extensions:
  policy:
    enabled: true  # default: false
```

### sources

List of policy sources. Each source is a directory of `.rego` files. The directory name should match the Rego `package` declaration inside the files.

#### Local Path

```yaml
extensions:
  policy:
    sources:
      - type: path
        path: terraform           # package terraform → data.terraform.deny/warn
      - type: path
        path: compliance          # package compliance → data.compliance.deny/warn
```

#### Git Repository

```yaml
extensions:
  policy:
    sources:
      - type: git
        url: https://github.com/org/terraform-policies.git
        ref: main                # Branch, tag, or commit SHA
```

#### OCI Registry

```yaml
extensions:
  policy:
    sources:
      - type: oci
        url: oci://ghcr.io/org/policies:v1.0
```

### namespaces

Rego package namespaces to evaluate. TerraCi queries `data.<namespace>.deny` and `data.<namespace>.warn` for each namespace.

```yaml
extensions:
  policy:
    namespaces:
      - terraform              # data.terraform.deny, data.terraform.warn
      - compliance             # data.compliance.deny, data.compliance.warn
```

Default: `["terraform"]`

Multiple namespaces allow separating concerns — security rules in `terraform`, cost rules in `compliance`, etc.

### decisions

Actions for OPA decisions:

| Value | Description |
|-------|-------------|
| `block` | Fail the pipeline (exit code 1) — **default** |
| `warn` | Reclassify failures as warnings, continue (exit code 0) |
| `ignore` | Silently ignore failures |

```yaml
extensions:
  policy:
    decisions:
      deny: block  # default
      warn: warn   # default
```

`decisions.deny` applies to Rego `deny` rules. `decisions.warn` applies to Rego `warn` rules.

### source_cache_dir

Directory for caching downloaded policies (git/OCI sources).

```yaml
extensions:
  policy:
    source_cache_dir: .terraci/policies  # default
```

### overrides

Override policy settings for specific modules using `**` glob patterns:

```yaml
extensions:
  policy:
    enabled: true
    decisions:
      deny: block

    overrides:
      # Sandbox: reclassify failures as warnings (don't block)
      - match: "**/sandbox/**"
        decisions:
          deny: warn

      # Legacy: skip policy checks entirely
      - match: "legacy/**"
        enabled: false

      # Production: add compliance namespace
      - match: "**/prod/**"
        namespaces:
          - terraform
          - compliance
```

#### Glob patterns

| Pattern | Matches | Does NOT match |
|---------|---------|---------------|
| `**/sandbox/**` | `platform/sandbox/eu-central-1/test` | `platform/stage/eu-central-1/app` |
| `legacy/**` | `legacy/old/eu-central-1/db` | `platform/legacy/module` |
| `**/prod/**` | `platform/prod/eu-central-1/vpc` | `platform/stage/eu-central-1/vpc` |
| `*/stage/*/*` | `platform/stage/eu-central-1/vpc` | `platform/stage/eu-central-1/ec2/sub` |

- `**` matches any number of path segments (including zero)
- `*` matches a single path segment

#### Override behavior

- **`decisions.deny: warn`** — deny rule violations are reclassified as warnings (appear in output but don't block)
- **`enabled: false`** — module is skipped entirely, no evaluation
- **`namespaces: [...]`** — replaces the namespace list for matching modules
- Multiple overrides can match the same module — applied in order

## Writing Policies

Policies use OPA v1 Rego syntax with `import rego.v1`.

### Deny Rules (block deployment)

```rego
package terraform

import rego.v1

# METADATA
# description: Deny public S3 buckets
# entrypoint: true
deny contains msg if {
    some resource in input.plan.resource_changes
    resource.type == "aws_s3_bucket"
    not "delete" in resource.change.actions
    resource.change.after.acl == "public-read"
    msg := sprintf("S3 bucket '%s' must not be public", [resource.name])
}
```

### Warn Rules (allow with warning)

```rego
warn contains msg if {
    some resource in input.plan.resource_changes
    resource.type == "aws_s3_bucket"
    not "delete" in resource.change.actions
    not has_versioning(resource)
    msg := sprintf("S3 bucket '%s' should have versioning", [resource.name])
}

has_versioning(resource) if {
    some v in resource.change.after.versioning
    v.enabled == true
}
```

### Key Rego patterns

| Pattern | Use |
|---------|-----|
| `some resource in input.plan.resource_changes` | Iterate resources (not `[_]`) |
| `"create" in resource.change.actions` | Check action membership |
| `not "delete" in resource.change.actions` | Negated membership |
| `resource.type in taggable_types` | Check if value in list |
| `deny contains msg if { ... }` | Deny rule (blocks pipeline) |
| `warn contains msg if { ... }` | Warn rule (doesn't block) |

::: tip Linting
Use [Regal](https://docs.styra.com/regal) to lint your policies: `regal lint terraform/ compliance/`
:::

### Multiple namespaces

Separate policies by concern — each source directory is a Rego package:

```
terraform/          → package terraform    (security baseline)
  tags.rego
  s3.rego
  ec2.rego
compliance/         → package compliance   (cost controls)
  cost.rego
```

```yaml
extensions:
  policy:
    sources:
      - type: path
        path: terraform
      - type: path
        path: compliance
    namespaces:
      - terraform
      - compliance
```

### Input Structure

Policies receive an envelope with TerraCi context and the raw Terraform plan JSON under `input.plan`:

```json
{
  "terraci": {
    "module": {
      "path": "platform/prod/eu-central-1/vpc",
      "components": {
        "service": "platform",
        "environment": "prod",
        "region": "eu-central-1",
        "module": "vpc"
      }
    },
    "policy": { "namespaces": ["terraform"] },
    "plan": { "path": "platform/prod/eu-central-1/vpc/plan.json" }
  },
  "plan": {
    "format_version": "1.2",
    "resource_changes": [
      {
        "type": "aws_s3_bucket",
        "name": "example",
        "change": {
          "actions": ["create"],
          "before": null,
          "after": {
            "bucket": "my-bucket",
            "acl": "private",
            "tags": { "Environment": "stage" }
          }
        }
      }
    ]
  }
}
```

See [Terraform JSON output format](https://developer.hashicorp.com/terraform/internals/json-format) for complete schema.

## Generated Pipeline

When policy checks are enabled, TerraCi adds a `policy-check` stage between plan and apply:

**GitLab CI:**
```yaml
stages:
  - deploy-0
  - policy-check
  - deploy-1

policy-check:
  stage: policy-check
  script:
    - terraci policy check --format text
  needs: [plan-vpc, plan-eks]
  artifacts:
    paths: [.terraci/policy-results.json]
```

**GitHub Actions:**
```yaml
jobs:
  policy-check:
    needs: [plan-vpc, plan-eks]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/download-artifact@v4
      - run: terraci policy check --format text
```

## Commands

```bash
# Prewarm/debug policy sources manually
terraci policy pull

# Check all modules with plan.json
terraci policy check

# Check specific module
terraci policy check --module platform/prod/eu-central-1/vpc

# JSON output
terraci policy check --format json

# Verbose output
terraci policy check -v
```

## MR/PR Integration

Policy results are included in the MR/PR comment:

```markdown
### ❌ Policy Check

**5** modules checked: ✅ **2** passed | ⚠️ **1** warned | ❌ **2** failed

<details>
<summary>❌ Failures (2)</summary>

**platform/stage/eu-central-1/bad:**
- `terraform`: S3 bucket 'public' must not have public-read ACL
- `compliance`: Instance 'web' uses expensive type 'p4d.24xlarge'

</details>
```

## Examples

See [examples/policy-checks](https://github.com/edelwud/terraci/tree/main/examples/policy-checks) for a complete working example with:

- Two namespaces (`terraform` + `compliance`)
- Five modules demonstrating pass, warn, fail, and skip statuses
- Overwrites for sandbox (warn) and legacy (disabled)
- Regal-compliant Rego policies

## See Also

- [Policy CLI](/cli/policy) — pull and check commands
