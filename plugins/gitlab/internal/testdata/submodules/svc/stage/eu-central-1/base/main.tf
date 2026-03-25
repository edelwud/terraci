# Base module - no dependencies

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "svc/stage/eu-central-1/base/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

output "vpc_id" {
  value = aws_vpc.main.id
}
