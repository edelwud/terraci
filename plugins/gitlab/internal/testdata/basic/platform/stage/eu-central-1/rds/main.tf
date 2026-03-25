# RDS module - depends on VPC

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/rds/terraform.tfstate"
    region = "eu-central-1"
  }
}

# Dependency on VPC
data "terraform_remote_state" "vpc" {
  backend = "s3"

  config = {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_db_instance" "main" {
  identifier     = "stage-rds"
  engine         = "postgres"
  engine_version = "14"
  instance_class = "db.t3.medium"

  db_subnet_group_name = aws_db_subnet_group.main.name

  allocated_storage = 20
  storage_type      = "gp2"

  username = "admin"
  password = "changeme"

  tags = {
    Name        = "stage-rds"
    Environment = "stage"
  }
}

resource "aws_db_subnet_group" "main" {
  name       = "stage-rds-subnet-group"
  subnet_ids = data.terraform_remote_state.vpc.outputs.subnet_ids
}

output "db_endpoint" {
  value = aws_db_instance.main.endpoint
}

output "db_name" {
  value = aws_db_instance.main.identifier
}
