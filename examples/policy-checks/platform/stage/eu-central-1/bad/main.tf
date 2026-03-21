terraform {
  backend "s3" {}
}

provider "aws" {
  region = "eu-central-1"
}

# Depends on VPC
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "example-state"
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

# Intentionally non-compliant resources for policy testing

resource "aws_instance" "no_tags" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"
  # Missing: tags, metadata_options, subnet_id
}

resource "aws_s3_bucket" "public" {
  bucket = "example-public-bucket"
  # Intentionally public — should trigger deny
}
