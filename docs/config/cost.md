---
title: Cost Estimation
description: "AWS cost estimation: pricing API cache, supported resources, and MR comment integration"
outline: deep
---

# Cost Estimation

TerraCi can estimate the monthly cost impact of infrastructure changes by analyzing Terraform plans against AWS pricing data. Cost estimates are calculated per module and displayed alongside plan results in MR comments.

## Basic Configuration

```yaml
cost:
  enabled: true
  show_in_comment: true
```

## Configuration Options

### enabled

Enable or disable cost estimation globally.

```yaml
cost:
  enabled: true  # default: false
```

### cache_dir

Directory to cache AWS pricing data fetched from the Bulk Pricing API. Caching avoids repeated API calls and speeds up subsequent runs.

```yaml
cost:
  cache_dir: ~/.terraci/pricing  # default
```

### cache_ttl

How long cached pricing data remains valid before being re-fetched.

```yaml
cost:
  cache_ttl: "24h"  # default
```

Accepts Go duration strings: `"1h"`, `"30m"`, `"72h"`, etc.

### show_in_comment

Include cost estimates in the MR/PR comment table alongside plan results and policy checks.

```yaml
cost:
  show_in_comment: true  # default: true
```

## How It Works

1. After `terraform plan` completes, TerraCi reads the `plan.json` file from each module directory.
2. Resource changes are extracted and matched against registered AWS resource handlers.
3. Pricing is fetched from the AWS Bulk Pricing API and cached locally according to `cache_ttl`.
4. Per-resource hourly and monthly costs are calculated for both the before and after states.
5. Results are aggregated into a per-module cost summary with before/after/diff values.

Cost estimation requires `plan.json` (produced by `terraform show -json`) to be present in module directories. This is generated automatically when `plan_enabled: true` in the pipeline configuration (GitLab or GitHub).

## Supported AWS Resources

| Category | Resources |
|----------|-----------|
| **EC2** | Instances, EBS volumes, Elastic IPs, NAT Gateways |
| **RDS** | Instances, clusters, cluster instances |
| **ELB** | Application Load Balancers, Classic Load Balancers |
| **ElastiCache** | Clusters, replication groups |
| **EKS** | Clusters, node groups |
| **Serverless** | Lambda, DynamoDB, SQS, SNS, Secrets Manager |
| **Storage** | S3 buckets, VPC endpoints |

Each resource type has a dedicated handler that maps Terraform resource attributes to the corresponding pricing dimensions.

## MR/PR Integration

When `show_in_comment` is enabled, cost estimates appear in the MR/PR comment table. Each module shows its monthly cost difference:

```markdown
### 💰 Cost Estimation

| Module | Monthly Before | Monthly After | Diff |
|--------|---------------|---------------|------|
| platform/prod/eu-central-1/vpc | $120.50 | $185.30 | +$64.80 |
| platform/prod/eu-central-1/eks | $450.00 | $450.00 | $0.00 |
| **Total** | **$570.50** | **$635.30** | **+$64.80** |
```

## Full Example

```yaml
cost:
  enabled: true
  cache_dir: ~/.terraci/pricing
  cache_ttl: "24h"
  show_in_comment: true

# Works with either provider:
gitlab:
  plan_enabled: true
  mr:
    comment:
      enabled: true

# Or with GitHub:
# github:
#   plan_enabled: true
#   pr:
#     comment:
#       enabled: true
```

This configuration enables cost estimation with default caching, and displays the results in MR/PR comments alongside plan output.

## CLI Command

Run cost estimation directly from the command line:

```bash
# Estimate all modules with plan.json
terraci cost

# Estimate a single module
terraci cost --module platform/prod/eu-central-1/rds

# JSON output
terraci cost --output json

# Verbose — shows per-resource breakdown and pricing cache info
terraci cost -v
```

The `terraci cost` command scans for `plan.json` files, fetches pricing data, and outputs per-module cost estimates. Pricing cache location and TTL expiration are shown in the output.

> **Note:** `terraci cost` requires `cost.enabled: true` in your `.terraci.yaml`.

In CI pipelines, cost estimation runs automatically as part of the `terraci summary` command (which posts MR/PR comments). Use `terraci cost` for local development and ad-hoc cost checks.

## Examples

See [examples/cost-estimation](https://github.com/edelwud/terraci/tree/main/examples/cost-estimation) for a working example with VPC, EKS, and RDS modules.

## See Also

- [Merge Request Integration](/config/gitlab-mr) — MR comments with plan summaries and cost estimates
- [Pipeline Generation Guide](/guide/pipeline-generation) — end-to-end guide for generating CI pipelines
