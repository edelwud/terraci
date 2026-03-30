package testutil

import (
	"fmt"
	"testing"
)

func BuildParserBenchmarkModule(tb testing.TB, dir string, fileCount int) {
	tb.Helper()

	for i := range fileCount {
		WriteFile(tb, dir, fmt.Sprintf("bench_%02d.tf", i), fmt.Sprintf(`
locals {
  service_%02d = "platform"
  env_%02d     = "prod"
}

variable "region_%02d" {
  default = "us-east-1"
}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

data "terraform_remote_state" "vpc_%02d" {
  backend = "s3"
  config = {
    bucket = "state-bucket"
    key    = "platform/prod/us-east-1/vpc/terraform.tfstate"
    region = "us-east-1"
  }
}

module "vpc_%02d" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`, i, i, i, i, i))
	}
}

func BuildModuleParseBenchmarkModule(tb testing.TB, dir string, fileCount int) {
	tb.Helper()

	for i := range fileCount {
		WriteFile(tb, dir, fmt.Sprintf("bench_%02d.tf", i), fmt.Sprintf(`
locals {
  service_%02d = "platform"
}

variable "region_%02d" {
  default = "us-east-1"
}

data "terraform_remote_state" "dep_%02d" {
  backend = "s3"
  config = {
    bucket = "state"
    key    = "platform/prod/us-east-1/vpc/terraform.tfstate"
  }
}

module "lib_%02d" {
  source = "../_modules/lib"
}
`, i, i, i, i))
	}
}
