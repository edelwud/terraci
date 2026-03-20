terraform {
  backend "s3" {}
}

provider "aws" {
  region = "eu-central-1"
}

data "terraform_remote_state" "eks" {
  backend = "s3"
  config = {
    bucket = "my-state"
    key    = "platform/stage/eu-central-1/eks/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "kubernetes_namespace" "app" {
  metadata {
    name = "my-app"
  }
}
