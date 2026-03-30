package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var benchParsedModule *ParsedModule

func BenchmarkParseModule(b *testing.B) {
	for _, tc := range []struct {
		name      string
		fileCount int
	}{
		{name: "files=5", fileCount: 5},
		{name: "files=20", fileCount: 20},
		{name: "files=50", fileCount: 50},
	} {
		dir := b.TempDir()
		buildParserBenchModule(b, dir, tc.fileCount)
		parser := NewParser(nil)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				parsed, err := parser.ParseModule(context.Background(), dir)
				if err != nil {
					b.Fatalf("ParseModule() error = %v", err)
				}
				benchParsedModule = parsed
			}
		})
	}
}

func buildParserBenchModule(tb testing.TB, dir string, fileCount int) {
	tb.Helper()

	for i := range fileCount {
		content := fmt.Sprintf(`
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
`, i, i, i, i, i)

		path := filepath.Join(dir, fmt.Sprintf("bench_%02d.tf", i))
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			tb.Fatalf("write %s: %v", path, err)
		}
	}
}
