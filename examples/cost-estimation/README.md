# Cost Estimation Example

AWS cost estimation for Terraform plans — see monthly costs and diffs in MR/PR comments.

## Structure

```
cost-estimation/
├── .terraci.yaml                              # Config with cost: enabled
└── platform/prod/eu-central-1/
    ├── vpc/                                   # NAT Gateway, EIP
    ├── eks/  (← vpc)                          # EKS cluster + m5.xlarge node group
    └── rds/  (← vpc)                          # RDS db.r6g.xlarge + ElastiCache
```

## How It Works

TerraCi estimates costs by:

1. Parsing `plan.json` (output of `terraform show -json plan.tfplan`)
2. Fetching AWS pricing data from the [Bulk Pricing API](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/price-changes.html)
3. Matching each resource to a cost handler (EC2, RDS, EKS, S3, Lambda, etc.)
4. Calculating monthly cost before/after and the diff

Pricing data is cached locally (default: `.terraci/pricing`, TTL: 24h).

## Configuration

```yaml
cost:
  enabled: true
  show_in_comment: true       # Include cost table in MR/PR comments
  cache_dir: .terraci/pricing # Local pricing cache
  cache_ttl: "24h"            # Re-fetch after this duration
```

No AWS credentials required — pricing data is public.

## Try It

```bash
cd examples/cost-estimation

# See dependency graph
terraci graph --format levels

# Generate pipeline (cost estimation runs during plan stage)
terraci generate --dry-run

# Validate structure
terraci validate
```

In CI, cost estimation happens automatically after `terraform plan`:

```bash
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
# TerraCi reads plan.json and estimates costs
```

## MR/PR Comment Output

When `show_in_comment: true`, the plan summary includes a cost column:

```
| Status | Module | Summary | Cost |
|:------:|--------|---------|------|
| 🔄 | platform/prod/eu-central-1/eks | +2 ~1 | $450.00 +$150.00 → $600.00 |
| 🔄 | platform/prod/eu-central-1/rds | +1    | $0 +$380.50 → $380.50 |
| ✅ | platform/prod/eu-central-1/vpc | No changes | $95.00 |
```

## Supported Resources

TerraCi estimates costs for these AWS resource types:

| Category | Resources |
|----------|-----------|
| **Compute** | EC2 instances, EBS volumes, EIPs, NAT Gateways |
| **Containers** | EKS clusters, EKS node groups, ECS/Fargate |
| **Database** | RDS instances/clusters, Aurora, ElastiCache |
| **Load Balancing** | ALB, NLB, Classic LB |
| **Serverless** | Lambda (provisioned concurrency), DynamoDB (provisioned) |
| **Storage** | S3 (fixed costs), EBS optimization, VPC endpoints |
| **Other** | Secrets Manager, KMS keys, Route53 zones, CloudWatch alarms |

Resources with purely usage-based pricing (S3 storage, Lambda requests, SQS) show `$0` since actual usage is unknown at plan time.

## How Costs Are Calculated

For each resource change in the plan:

1. **Resource handler** matches the resource type (e.g., `aws_db_instance` → RDS handler)
2. **Price lookup** extracts attributes (instance class, region, engine) and queries cached pricing
3. **Cost calculation** returns hourly and monthly estimates
4. **Aggregation** produces per-module cost with before/after/diff

Example for `aws_db_instance`:
- Instance: `db.r6g.xlarge` → ~$0.48/hr → ~$350/mo
- Storage: 100GB gp3 → ~$8/mo
- Multi-AZ: doubles instance cost → ~$700/mo total

## See Also

- [Cost configuration reference](/config/cost)
- [examples/policy-checks](../policy-checks/) — combine cost estimation with OPA policies
