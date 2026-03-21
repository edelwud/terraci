terraform {
  backend "s3" {}
}

provider "aws" {
  region = "eu-central-1"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name        = "stage-vpc"
    Environment = "stage"
    Project     = "example"
    Owner       = "devops"
  }
}

resource "aws_s3_bucket" "logs" {
  bucket = "example-vpc-logs"

  # Has tags but NO versioning and NO encryption → should trigger warnings

  tags = {
    Environment = "stage"
    Project     = "example"
    Owner       = "devops"
  }
}
