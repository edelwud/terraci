terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "platform/prod/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}
