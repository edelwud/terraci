# Module A - depends on C (creates cycle: a -> c -> b -> a)

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "svc/env/region/a/terraform.tfstate"
    region = "eu-central-1"
  }
}

data "terraform_remote_state" "c" {
  backend = "s3"

  config = {
    bucket = "terraform-state"
    key    = "svc/env/region/c/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "null_resource" "a" {
  triggers = {
    c_output = data.terraform_remote_state.c.outputs.value
  }
}

output "value" {
  value = "a"
}
