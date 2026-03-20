package plan

// --- Sample plan JSON fixtures ---

const samplePlanJSON = `{
  "format_version": "1.2",
  "terraform_version": "1.6.0",
  "resource_changes": [
    {
      "address": "aws_instance.web", "mode": "managed", "type": "aws_instance", "name": "web",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["update"],
        "before": {"ami": "ami-12345", "instance_type": "t2.micro", "tags": {"Name": "old-name"}},
        "after":  {"ami": "ami-12345", "instance_type": "t2.small", "tags": {"Name": "new-name"}},
        "after_unknown": {}, "before_sensitive": {}, "after_sensitive": {}
      }
    },
    {
      "address": "aws_s3_bucket.data", "mode": "managed", "type": "aws_s3_bucket", "name": "data",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {"bucket": "my-data-bucket", "tags": {}},
        "after_unknown": {"id": true}, "before_sensitive": {}, "after_sensitive": {}
      }
    },
    {
      "address": "aws_instance.old", "mode": "managed", "type": "aws_instance", "name": "old",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["delete"],
        "before": {"ami": "ami-old", "instance_type": "t2.micro"},
        "after": null,
        "after_unknown": {}, "before_sensitive": {}, "after_sensitive": {}
      }
    }
  ]
}`

const samplePlanJSONNoChanges = `{
  "format_version": "1.2", "terraform_version": "1.6.0",
  "resource_changes": [{
    "address": "aws_instance.web", "mode": "managed", "type": "aws_instance", "name": "web",
    "change": {"actions": ["no-op"], "before": {"ami": "ami-12345"}, "after": {"ami": "ami-12345"}}
  }]
}`

const samplePlanJSONReplace = `{
  "format_version": "1.2", "terraform_version": "1.6.0",
  "resource_changes": [{
    "address": "aws_instance.web", "mode": "managed", "type": "aws_instance", "name": "web",
    "change": {
      "actions": ["delete", "create"],
      "before": {"ami": "ami-old"}, "after": {"ami": "ami-new"},
      "replace_paths": [["ami"]]
    }
  }]
}`

const samplePlanJSONWithModule = `{
  "format_version": "1.2", "terraform_version": "1.6.0",
  "resource_changes": [
    {
      "address": "module.vpc.aws_vpc.main", "module_address": "module.vpc",
      "mode": "managed", "type": "aws_vpc", "name": "main",
      "change": {"actions": ["create"], "before": null, "after": {"cidr_block": "10.0.0.0/16"}}
    },
    {
      "address": "module.vpc.module.subnets.aws_subnet.private[0]",
      "module_address": "module.vpc.module.subnets",
      "mode": "managed", "type": "aws_subnet", "name": "private", "index": 0,
      "change": {"actions": ["create"], "before": null, "after": {"cidr_block": "10.0.1.0/24"}}
    }
  ]
}`

const samplePlanJSONSensitive = `{
  "format_version": "1.2", "terraform_version": "1.6.0",
  "resource_changes": [{
    "address": "aws_db_instance.main", "mode": "managed", "type": "aws_db_instance", "name": "main",
    "change": {
      "actions": ["update"],
      "before": {"password": "old-secret"}, "after": {"password": "new-secret"},
      "before_sensitive": {"password": true}, "after_sensitive": {"password": true}
    }
  }]
}`
