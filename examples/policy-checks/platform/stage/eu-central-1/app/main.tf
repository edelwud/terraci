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

resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"
  subnet_id     = data.terraform_remote_state.vpc.outputs.subnet_id

  metadata_options {
    http_tokens = "required"
  }

  tags = {
    Name        = "web-server"
    Environment = "stage"
    Project     = "example"
    Owner       = "devops"
  }
}

resource "aws_s3_bucket" "data" {
  bucket = "example-data-bucket"

  tags = {
    Environment = "stage"
    Project     = "example"
    Owner       = "devops"
  }
}
