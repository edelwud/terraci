# Module C - depends on B

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "svc/env/region/c/terraform.tfstate"
    region = "eu-central-1"
  }
}

data "terraform_remote_state" "b" {
  backend = "s3"

  config = {
    bucket = "terraform-state"
    key    = "svc/env/region/b/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "null_resource" "c" {
  triggers = {
    b_output = data.terraform_remote_state.b.outputs.value
  }
}

output "value" {
  value = "c"
}
