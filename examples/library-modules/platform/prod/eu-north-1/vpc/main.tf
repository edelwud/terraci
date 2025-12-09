# VPC module - executable module with providers and state

terraform {
  required_version = ">= 1.0"

  backend "s3" {
    bucket = "my-terraform-state"
    key    = "platform/prod/eu-north-1/vpc/terraform.tfstate"
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

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "${local.service}-${local.environment}-${local.region}"
  cidr = "10.0.0.0/16"

  azs             = ["eu-north-1a", "eu-north-1b", "eu-north-1c"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]

  enable_nat_gateway = true
  single_nat_gateway = true

  tags = {
    Environment = local.environment
    Service     = local.service
    Region      = local.region
  }
}

output "vpc_id" {
  value = module.vpc.vpc_id
}

output "private_subnet_ids" {
  value = module.vpc.private_subnets
}

output "public_subnet_ids" {
  value = module.vpc.public_subnets
}
