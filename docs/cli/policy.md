# terraci policy

Commands for managing and running OPA policy checks against Terraform plans.

## Subcommands

### terraci policy pull

Download policies from configured sources.

```bash
terraci policy pull
```

This command:
1. Reads policy sources from `.terraci.yaml`
2. Downloads policies to the cache directory
3. Prepares policies for evaluation

**Example output:**
```
pulling policies...
  source: path:policies
  source: git:https://github.com/org/policies.git@main
pulled 2 policy sources to .terraci/policies
```

### terraci policy check

Run policy checks against Terraform plan JSON files.

```bash
terraci policy check [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--module` | `-m` | Check specific module path |
| `--output` | `-o` | Output format: `text` (default) or `json` |

**Examples:**

```bash
# Check all modules with plan.json files
terraci policy check

# Check specific module
terraci policy check --module platform/prod/eu-central-1/vpc

# Output as JSON
terraci policy check --output json
```

**Text output:**
```
Policy Check Results
====================

✅ platform/prod/eu-central-1/vpc
   0 failures, 0 warnings

⚠️ platform/prod/eu-central-1/ec2
   0 failures, 1 warning
   - terraform: Instance 'web' should have Environment tag

❌ platform/prod/eu-central-1/s3
   1 failure, 0 warnings
   - terraform: S3 bucket 'logs' must not be public

Summary: 3 modules checked
  Passed:  1
  Warned:  1
  Failed:  1
```

**JSON output:**
```json
{
  "total_modules": 3,
  "passed_modules": 1,
  "warned_modules": 1,
  "failed_modules": 1,
  "total_failures": 1,
  "total_warnings": 1,
  "results": [
    {
      "module": "platform/prod/eu-central-1/vpc",
      "failures": [],
      "warnings": [],
      "successes": 5
    },
    {
      "module": "platform/prod/eu-central-1/ec2",
      "failures": [],
      "warnings": [
        {
          "msg": "Instance 'web' should have Environment tag",
          "namespace": "terraform"
        }
      ],
      "successes": 3
    },
    {
      "module": "platform/prod/eu-central-1/s3",
      "failures": [
        {
          "msg": "S3 bucket 'logs' must not be public",
          "namespace": "terraform"
        }
      ],
      "warnings": [],
      "successes": 2
    }
  ]
}
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | All checks passed (or `on_failure: warn/ignore`) |
| 1 | Policy violations found (when `on_failure: block`) |
| 2 | Configuration or runtime error |

## Requirements

Policy checks require `plan.json` files in module directories. Generate them with:

```bash
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
```

Or in GitLab CI:

```yaml
plan-module:
  script:
    - terraform init
    - terraform plan -out=plan.tfplan
    - terraform show -json plan.tfplan > plan.json
  artifacts:
    paths:
      - "**/plan.json"
```

## Configuration

See [Policy Configuration](/config/policy) for full configuration options.

Minimal example:

```yaml
policy:
  enabled: true
  sources:
    - path: policies
  namespaces:
    - terraform
  on_failure: block
```

## See Also

- [Policy Configuration](/config/policy) - Full configuration reference
- [examples/policy-checks](https://github.com/edelwud/terraci/tree/main/examples/policy-checks) - Example policies
