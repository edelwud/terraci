# Module B - depends on A

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "svc/env/region/b/terraform.tfstate"
    region = "eu-central-1"
  }
}

data "terraform_remote_state" "a" {
  backend = "s3"

  config = {
    bucket = "terraform-state"
    key    = "svc/env/region/a/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "null_resource" "b" {
  triggers = {
    a_output = data.terraform_remote_state.a.outputs.value
  }
}

output "value" {
  value = "b"
}
