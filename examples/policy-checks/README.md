# Policy Checks Example

OPA-based policy enforcement for Terraform plans — block violations, warn about issues, skip legacy modules.

## Structure

```
policy-checks/
├── .terraci.yaml                           # Config with two policy sources + overwrites
├── terraform/                              # "terraform" namespace policies
│   ├── tags.rego                           #   Required tags on all taggable resources
│   ├── s3.rego                             #   S3 encryption, ACL, versioning
│   └── ec2.rego                            #   IMDSv2, public IP, instance sizes
├── compliance/                             # "compliance" namespace policies
│   └── cost.rego                           #   Expensive instance types, cost approval
├── platform/
│   ├── stage/eu-central-1/
│   │   ├── vpc/                            # ❌ fail — S3 bucket without encryption
│   │   ├── app/                            # ✅ pass — all policies satisfied
│   │   └── bad/                            # ❌ fail — missing tags, public ACL, GPU instance
│   └── sandbox/eu-central-1/
│       └── test/                           # ⚠️ warn — failures reclassified (overwrite)
└── legacy/old/eu-central-1/
    └── db/                                 # ✅ skip — policy checks disabled (overwrite)
```

## Try It

```bash
cd examples/policy-checks

# Run policy checks
terraci policy check -v

# Check a single module
terraci policy check --module platform/stage/eu-central-1/bad -v

# JSON output
terraci policy check --output json

# See dependency graph
terraci graph --format levels
```

Expected output:

```
policy check summary   total=5 passed=2 warned=1 failed=2

module: platform/sandbox/eu-central-1/test   status=warn     ← on_failure: warn overwrite
module: platform/stage/eu-central-1/bad      status=fail     ← 10 failures from terraform + compliance
module: platform/stage/eu-central-1/vpc      status=fail     ← S3 encryption missing
                                                              ← app passed, legacy skipped
```

## Configuration

```yaml
plugins:
  policy:
    enabled: true

    # Multiple policy sources — each directory is a Rego package
    sources:
      - path: terraform       # package terraform → deny/warn rules
      - path: compliance      # package compliance → cost rules

    # Evaluate both namespaces
    namespaces:
      - terraform
      - compliance

    on_failure: block         # Default: block pipeline on deny violations
    on_warning: warn          # Default: continue with warnings

    # Per-module overwrites using ** glob patterns
    overwrites:
      - match: "**/sandbox/**"
        on_failure: warn      # Sandbox: reclassify failures → warnings

      - match: "legacy/**"
        enabled: false        # Legacy: skip policy checks entirely
```

### Overwrites

`**` matches any number of path segments:

| Pattern | Matches | Does NOT match |
|---------|---------|---------------|
| `**/sandbox/**` | `platform/sandbox/eu-central-1/test` | `platform/stage/eu-central-1/app` |
| `legacy/**` | `legacy/old/eu-central-1/db` | `platform/legacy/something` |
| `**/prod/**` | `platform/prod/eu-central-1/vpc` | `platform/stage/eu-central-1/vpc` |

When `on_failure: warn` is set via overwrite, deny rule violations are **reclassified as warnings** — they appear in the output but don't block the pipeline.

When `enabled: false` is set, the module is **skipped entirely** — no evaluation happens.

## Policy Namespaces

Each source directory maps to a Rego `package` name. The directory name must match the package declaration:

```
terraform/tags.rego    → package terraform    ← evaluated when "terraform" in namespaces
compliance/cost.rego   → package compliance   ← evaluated when "compliance" in namespaces
```

### terraform namespace

Security and compliance baseline:

| Policy | Type | Rule |
|--------|------|------|
| `tags.rego` | deny | Resources missing required tags (Environment, Project, Owner) |
| `tags.rego` | warn | Resources with empty tag values |
| `s3.rego` | deny | Public ACL (public-read, public-read-write) |
| `s3.rego` | deny | Missing server-side encryption |
| `s3.rego` | warn | Missing versioning |
| `ec2.rego` | deny | Public IP in production |
| `ec2.rego` | deny | Missing IMDSv2 |
| `ec2.rego` | warn | Missing subnet_id (default VPC) |
| `ec2.rego` | warn | Large instance types (GPU, memory-optimized) |

### compliance namespace

Cost controls:

| Policy | Type | Rule |
|--------|------|------|
| `cost.rego` | deny | Expensive instance types without `CostApproved=true` tag |
| `cost.rego` | warn | RDS multi-AZ deployment (cost doubles) |
| `cost.rego` | warn | EBS volumes > 500GB |

## Writing Policies

Policies use OPA v1 Rego syntax with `import rego.v1`:

```rego
package terraform

import rego.v1

# METADATA
# description: Deny public S3 buckets
# entrypoint: true
deny contains msg if {
    some resource in input.resource_changes
    resource.type == "aws_s3_bucket"
    not "delete" in resource.change.actions
    resource.change.after.acl == "public-read"
    msg := sprintf("S3 bucket '%s' must not be public", [resource.name])
}
```

Key patterns:
- `some resource in input.resource_changes` — iterate resources (not `[_]`)
- `"create" in resource.change.actions` — check membership (not `== "create"`)
- `not "delete" in resource.change.actions` — negated membership
- `deny contains msg if` — deny rules block; `warn contains msg if` — warn rules don't
- `# METADATA` + `# entrypoint: true` — must be directly attached to the first rule

Lint your policies with [Regal](https://docs.styra.com/regal): `regal lint terraform/ compliance/`

## Pipeline Integration

```yaml
# .gitlab-ci.yml
generate:
  stage: generate
  image: ghcr.io/edelwud/terraci:latest
  script:
    - terraci generate --changed-only -o terraform.yml
  artifacts:
    paths: [terraform.yml]

deploy:
  stage: deploy
  trigger:
    include:
      - artifact: terraform.yml
        job: generate
```

The generated pipeline includes a `policy-check` stage between plan and apply. If `on_failure: block` and any deny rules fire, the pipeline stops.

## Policy Sources

```yaml
plugins:
  policy:
    sources:
      # Local directory
      - path: terraform

      # Git repository
      - git: https://github.com/org/terraform-policies.git
        ref: main

      # OCI registry
      - oci: oci://ghcr.io/org/policies:v1.0
```
