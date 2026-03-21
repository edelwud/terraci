terraform {
  backend "s3" {}
}

provider "aws" {
  region = "eu-central-1"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  tags       = { Name = "prod-vpc" }
}

resource "aws_nat_gateway" "main" {
  allocation_id = "eipalloc-12345"
  subnet_id     = "subnet-12345"
  tags          = { Name = "prod-nat" }
}

resource "aws_eip" "nat" {
  domain = "vpc"
  tags   = { Name = "prod-nat-eip" }
}
