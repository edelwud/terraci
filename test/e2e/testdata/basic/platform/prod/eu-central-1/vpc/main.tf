# VPC module for prod environment - no dependencies

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "platform/prod/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_vpc" "main" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name        = "prod-vpc"
    Environment = "prod"
  }
}

output "vpc_id" {
  value = aws_vpc.main.id
}

output "vpc_cidr" {
  value = aws_vpc.main.cidr_block
}
