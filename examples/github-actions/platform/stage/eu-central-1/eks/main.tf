terraform {
  backend "s3" {}
}

provider "aws" {
  region = "eu-central-1"
}

data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "my-state"
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_eks_cluster" "main" {
  name     = "stage-eks"
  role_arn = "arn:aws:iam::role/eks"

  vpc_config {
    subnet_ids = data.terraform_remote_state.vpc.outputs.subnet_ids
  }
}
