# EC2/Worker submodule - depends on base

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "svc/stage/eu-central-1/ec2/worker/terraform.tfstate"
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

resource "aws_instance" "worker" {
  ami           = "ami-12345678"
  instance_type = "t3.large"
  subnet_id     = data.terraform_remote_state.base.outputs.vpc_id

  tags = {
    Name = "worker-server"
    Type = "worker"
  }
}

output "instance_id" {
  value = aws_instance.worker.id
}
