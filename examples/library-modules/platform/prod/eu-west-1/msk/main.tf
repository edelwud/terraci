# MSK module for eu-west-1 - another executable module using the same library modules

terraform {
  required_version = ">= 1.0"

  backend "s3" {
    bucket = "my-terraform-state"
    key    = "platform/prod/eu-west-1/msk/terraform.tfstate"
    region = "eu-central-1"
  }
}

provider "aws" {
  region = "eu-west-1"
}

locals {
  service     = "platform"
  environment = "prod"
  region      = "eu-west-1"
}

# This module doesn't have a VPC in this example, but demonstrates
# that library module changes affect all modules using them

# Using the kafka library module
module "kafka" {
  source = "../../../../_modules/kafka"

  cluster_name       = "${local.service}-${local.environment}-${local.region}"
  subnet_ids         = ["subnet-12345", "subnet-67890", "subnet-abcde"]
  security_group_ids = ["sg-12345"]

  broker_count    = 3
  instance_type   = "kafka.m5.large"
  ebs_volume_size = 200

  auto_create_topics  = false
  replication_factor  = 3
  min_insync_replicas = 2
  default_partitions  = 12
  log_retention_hours = 336

  tags = {
    Environment = local.environment
    Service     = local.service
    Region      = local.region
  }
}

# Using the kafka_acl library module
module "analytics_acls" {
  source = "../../../../_modules/kafka_acl"

  cluster_arn = module.kafka.cluster_arn
  principal   = "User:analytics-service"

  read_topics  = ["events", "metrics", "logs"]
  write_topics = ["processed-events"]

  consumer_group = "analytics-consumer-group"
}

output "cluster_arn" {
  value = module.kafka.cluster_arn
}

output "bootstrap_brokers_tls" {
  value = module.kafka.bootstrap_brokers_tls
}
