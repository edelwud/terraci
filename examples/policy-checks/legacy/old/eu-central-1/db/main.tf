terraform {
  backend "s3" {}
}

provider "aws" {
  region = "eu-central-1"
}

resource "aws_instance" "legacy" {
  ami           = "ami-old"
  instance_type = "m4.large"
}
