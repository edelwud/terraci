terraform {
  required_version = "> 1.0"
}

resource "null_resource" "vpc" {}

output "vpc_id" {
  value = "vpc-stage"
}
