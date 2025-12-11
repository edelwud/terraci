# EKS module - depends on VPC

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/eks/terraform.tfstate"
    region = "eu-central-1"
  }
}

# Dependency on VPC
data "terraform_remote_state" "vpc" {
  backend = "s3"

  config = {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_eks_cluster" "main" {
  name     = "stage-eks"
  role_arn = aws_iam_role.eks.arn

  vpc_config {
    subnet_ids = data.terraform_remote_state.vpc.outputs.subnet_ids
  }

  tags = {
    Name        = "stage-eks"
    Environment = "stage"
  }
}

resource "aws_iam_role" "eks" {
  name = "stage-eks-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "eks.amazonaws.com"
      }
    }]
  })
}

output "cluster_endpoint" {
  value = aws_eks_cluster.main.endpoint
}

output "cluster_name" {
  value = aws_eks_cluster.main.name
}
