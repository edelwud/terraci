terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "platform/prod/eu-central-1/eks/terraform.tfstate"
    region = "eu-central-1"
  }
}

data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "terraform-state"
    key    = "platform/prod/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_eks_cluster" "main" {
  name = "production"
  vpc_config {
    subnet_ids = data.terraform_remote_state.vpc.outputs.subnet_ids
  }
}
