terraform {
  required_version = "> 1.0"
}

resource "null_resource" "vpn_vpc" {}

output "vpc_id" {
  value = "vpc-vpn"
}
