# EC2/Web submodule - depends on base

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "svc/stage/eu-central-1/ec2/web/terraform.tfstate"
    region = "eu-central-1"
  }
}

data "terraform_remote_state" "base" {
  backend = "s3"

  config = {
    bucket = "terraform-state"
    key    = "svc/stage/eu-central-1/base/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_instance" "web" {
  ami           = "ami-12345678"
  instance_type = "t3.medium"
  subnet_id     = data.terraform_remote_state.base.outputs.vpc_id

  tags = {
    Name = "web-server"
    Type = "web"
  }
}

output "instance_id" {
  value = aws_instance.web.id
}
