# Submodules

TerraCi supports nested submodules at depth 5, allowing you to organize related infrastructure within a parent module.

## What Are Submodules?

Submodules are Terraform modules nested one level deeper than the standard pattern:

```
infrastructure/
└── platform/
    └── production/
        └── us-east-1/
            └── ec2/                    # Parent module (depth 4)
                ├── main.tf             # Parent's Terraform files
                ├── rabbitmq/           # Submodule (depth 5)
                │   └── main.tf
                └── redis/              # Submodule (depth 5)
                    └── main.tf
```

## Module Identification

| Path | Type | ID |
|------|------|-----|
| `platform/production/us-east-1/ec2` | Parent | `platform/production/us-east-1/ec2` |
| `platform/production/us-east-1/ec2/rabbitmq` | Submodule | `platform/production/us-east-1/ec2/rabbitmq` |
| `platform/production/us-east-1/ec2/redis` | Submodule | `platform/production/us-east-1/ec2/redis` |

## Configuration

Enable submodules in `.terraci.yaml`:

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5              # Allow depth 5 for submodules
  allow_submodules: true    # Enable submodule discovery
```

## Use Cases

### Grouped EC2 Instances

Group related EC2 instances under a parent:

```
ec2/
├── main.tf           # Shared security groups, IAM roles
├── rabbitmq/
│   └── main.tf       # RabbitMQ-specific EC2 instances
└── elasticsearch/
    └── main.tf       # Elasticsearch EC2 instances
```

### Database Clusters

Organize database types:

```
databases/
├── main.tf           # Shared VPC, subnets
├── postgresql/
│   └── main.tf       # PostgreSQL RDS
├── redis/
│   └── main.tf       # ElastiCache Redis
└── mongodb/
    └── main.tf       # DocumentDB
```

### Microservices

Group microservices by domain:

```
services/
├── auth/
│   └── main.tf
├── payments/
│   └── main.tf
└── notifications/
    └── main.tf
```

## Dependencies

Submodules can depend on:
- Their parent module
- Other submodules in the same parent
- Modules outside their parent

### Parent-Child Dependencies

```hcl
# In ec2/rabbitmq/main.tf
data "terraform_remote_state" "ec2_parent" {
  backend = "s3"
  config = {
    key = "platform/production/us-east-1/ec2/terraform.tfstate"
  }
}
```

### Sibling Dependencies

```hcl
# In ec2/app/main.tf
data "terraform_remote_state" "rabbitmq" {
  backend = "s3"
  config = {
    key = "platform/production/us-east-1/ec2/rabbitmq/terraform.tfstate"
  }
}
```

## Name Matching

TerraCi uses intelligent name matching for submodules:

| Remote State Name | Matches |
|-------------------|---------|
| `ec2_rabbitmq` | `ec2/rabbitmq` |
| `ec2-rabbitmq` | `ec2/rabbitmq` |
| `rabbitmq` | `ec2/rabbitmq` (within same context) |

## Generated Pipeline

Submodules appear as regular jobs:

```yaml
plan-platform-prod-us-east-1-ec2:
  stage: deploy-plan-0
  # ...

plan-platform-prod-us-east-1-ec2-rabbitmq:
  stage: deploy-plan-1
  needs:
    - apply-platform-prod-us-east-1-ec2
  # ...
```

## Parent Module Index

TerraCi maintains a parent-child relationship index:

```go
type Module struct {
    // ...
    Parent   *Module   // Reference to parent (for submodules)
    Children []*Module // References to children (for parents)
}
```

This enables:
- Querying all submodules of a parent
- Finding the parent of a submodule
- Building accurate dependency chains

## Best Practices

### 1. Keep Related Resources Together

Group resources that are deployed together:

```
app/
├── main.tf           # ECS cluster, load balancer
├── api/
│   └── main.tf       # API service
└── worker/
    └── main.tf       # Background worker service
```

### 2. Share Common Configuration

Put shared resources in the parent:

```hcl
# In ec2/main.tf
resource "aws_security_group" "shared" {
  name = "ec2-shared-sg"
}

output "shared_security_group_id" {
  value = aws_security_group.shared.id
}
```

```hcl
# In ec2/rabbitmq/main.tf
data "terraform_remote_state" "parent" {
  # ...
}

resource "aws_instance" "rabbitmq" {
  vpc_security_group_ids = [
    data.terraform_remote_state.parent.outputs.shared_security_group_id
  ]
}
```

### 3. Consistent Naming

Use consistent naming for submodules:

```
✓ ec2/rabbitmq
✓ ec2/redis
✓ ec2/elasticsearch

✗ ec2/rabbit-mq
✗ ec2/Redis
✗ ec2/es
```

### 4. Limit Nesting Depth

TerraCi supports only one level of submodules (depth 5). For deeper hierarchies, consider restructuring:

```
# Instead of:
platform/prod/us-east-1/services/backend/api/main.tf  # Too deep!

# Use:
platform/prod/us-east-1/backend-api/main.tf           # Flat
```

## Filtering Submodules

Include or exclude submodules with patterns:

```bash
# Only submodules
terraci generate --include "*/*//*/*/**"

# Exclude specific submodule
terraci generate --exclude "*/*/us-east-1/ec2/rabbitmq"

# Only parent modules (no submodules)
terraci generate --exclude "*/*/*/*/*"
```

## Troubleshooting

### Submodules Not Discovered

1. Check `max_depth` is set to 5
2. Verify `allow_submodules: true`
3. Ensure submodule directory contains `.tf` files

### Parent Not Linked

If submodule's parent isn't detected:

1. Verify parent exists at depth 4
2. Check parent contains `.tf` files
3. Run `terraci validate -v` to see discovery details
