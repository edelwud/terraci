# MSK module - executable module that uses library modules

terraform {
  required_version = ">= 1.0"

  backend "s3" {
    bucket = "my-terraform-state"
    key    = "platform/prod/eu-north-1/msk/terraform.tfstate"
    region = "eu-central-1"
  }
}

provider "aws" {
  region = "eu-north-1"
}

locals {
  service     = "platform"
  environment = "prod"
  region      = "eu-north-1"
}

# Dependency on VPC module via remote state
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "my-terraform-state"
    key    = "${local.service}/${local.environment}/${local.region}/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

# Security group for MSK
resource "aws_security_group" "msk" {
  name        = "${local.service}-${local.environment}-${local.region}-msk"
  description = "Security group for MSK cluster"
  vpc_id      = data.terraform_remote_state.vpc.outputs.vpc_id

  ingress {
    from_port   = 9092
    to_port     = 9092
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }

  ingress {
    from_port   = 9094
    to_port     = 9094
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# Using the kafka library module
module "kafka" {
  source = "../../../../_modules/kafka"

  cluster_name       = "${local.service}-${local.environment}-${local.region}"
  subnet_ids         = data.terraform_remote_state.vpc.outputs.private_subnet_ids
  security_group_ids = [aws_security_group.msk.id]

  broker_count    = 3
  instance_type   = "kafka.m5.large"
  ebs_volume_size = 100

  auto_create_topics  = false
  replication_factor  = 3
  min_insync_replicas = 2
  default_partitions  = 6
  log_retention_hours = 168

  tags = {
    Environment = local.environment
    Service     = local.service
    Region      = local.region
  }
}

# Using the kafka_acl library module for service access
module "service_acls" {
  source = "../../../../_modules/kafka_acl"

  cluster_arn = module.kafka.cluster_arn
  principal   = "User:service-account"

  read_topics  = ["events", "notifications"]
  write_topics = ["events"]

  consumer_group = "service-consumer-group"
}

output "cluster_arn" {
  value = module.kafka.cluster_arn
}

output "bootstrap_brokers_tls" {
  value = module.kafka.bootstrap_brokers_tls
}
