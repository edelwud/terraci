# VPC module - no dependencies (root level)

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name        = "stage-vpc"
    Environment = "stage"
  }
}

output "vpc_id" {
  value = aws_vpc.main.id
}

output "vpc_cidr" {
  value = aws_vpc.main.cidr_block
}
