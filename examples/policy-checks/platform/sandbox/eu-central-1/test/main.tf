terraform {
  backend "s3" {}
}

provider "aws" {
  region = "eu-central-1"
}

resource "aws_instance" "sandbox" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"
}
