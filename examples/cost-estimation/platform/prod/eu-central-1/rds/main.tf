terraform {
  backend "s3" {}
}

provider "aws" {
  region = "eu-central-1"
}

data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "example-state"
    key    = "platform/prod/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_db_instance" "postgres" {
  identifier     = "prod-postgres"
  engine         = "postgres"
  engine_version = "15.4"
  instance_class = "db.r6g.xlarge"

  allocated_storage     = 100
  max_allocated_storage = 500
  storage_type          = "gp3"
  storage_encrypted     = true

  multi_az = true

  db_subnet_group_name   = "prod-db-subnet"
  vpc_security_group_ids = ["sg-12345"]

  tags = { Name = "prod-postgres" }
}

resource "aws_elasticache_cluster" "redis" {
  cluster_id      = "prod-redis"
  engine          = "redis"
  node_type       = "cache.r6g.large"
  num_cache_nodes = 2

  tags = { Name = "prod-redis" }
}
