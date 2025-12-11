# App module - depends on EKS, RDS, and S3

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/app/terraform.tfstate"
    region = "eu-central-1"
  }
}

# Dependency on EKS
data "terraform_remote_state" "eks" {
  backend = "s3"

  config = {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/eks/terraform.tfstate"
    region = "eu-central-1"
  }
}

# Dependency on RDS
data "terraform_remote_state" "rds" {
  backend = "s3"

  config = {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/rds/terraform.tfstate"
    region = "eu-central-1"
  }
}

# Dependency on S3
data "terraform_remote_state" "s3" {
  backend = "s3"

  config = {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/s3/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "kubernetes_deployment" "app" {
  metadata {
    name = "stage-app"
  }

  spec {
    replicas = 2

    selector {
      match_labels = {
        app = "stage-app"
      }
    }

    template {
      metadata {
        labels = {
          app = "stage-app"
        }
      }

      spec {
        container {
          name  = "app"
          image = "myapp:latest"

          env {
            name  = "DATABASE_URL"
            value = data.terraform_remote_state.rds.outputs.db_endpoint
          }

          env {
            name  = "S3_BUCKET"
            value = data.terraform_remote_state.s3.outputs.bucket_name
          }
        }
      }
    }
  }
}

output "deployment_name" {
  value = kubernetes_deployment.app.metadata[0].name
}
