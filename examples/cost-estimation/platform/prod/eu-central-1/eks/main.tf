terraform {
  backend "s3" {}
}

provider "aws" {
  region = "eu-central-1"
}

data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "example-state"
    key    = "platform/prod/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_eks_cluster" "main" {
  name     = "prod-eks"
  role_arn = "arn:aws:iam::role/eks"

  vpc_config {
    subnet_ids = ["subnet-1", "subnet-2"]
  }

  tags = { Name = "prod-eks" }
}

resource "aws_eks_node_group" "workers" {
  cluster_name    = aws_eks_cluster.main.name
  node_group_name = "workers"
  node_role_arn   = "arn:aws:iam::role/eks-node"
  subnet_ids      = ["subnet-1", "subnet-2"]
  instance_types  = ["m5.xlarge"]

  scaling_config {
    desired_size = 3
    max_size     = 5
    min_size     = 1
  }

  tags = { Name = "prod-eks-workers" }
}
