terraform {
  required_version = "> 1.0"
  backend "s3" {
    bucket = "${local.service}-tf-state-bucket"
    key    = "${local.service}/${local.environment}/${local.region}/${local.scope}/${local.module}/terraform.tfstate"
    region = "eu-north-1"
  }
}

# Reference to VPC in same service/environment/region
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "${local.service}-tf-state-bucket"
    key    = "${local.service}/${local.environment}/${local.region}/vpc/terraform.tfstate"
    region = "eu-north-1"
  }
}

# Reference to VPN VPC in different environment/region (hardcoded)
data "terraform_remote_state" "vpn_vpc" {
  backend = "s3"
  config = {
    bucket = "${local.service}-tf-state-bucket"
    key    = "${local.service}/vpn/eu-north-1/vpc/terraform.tfstate"
    region = "eu-north-1"
  }
}

resource "null_resource" "db_migrate" {}
