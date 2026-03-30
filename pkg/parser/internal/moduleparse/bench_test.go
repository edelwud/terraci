package moduleparse

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/parser/model"
)

var benchModuleParseResult *model.ParsedModule

func BenchmarkRun(b *testing.B) {
	for _, tc := range []struct {
		name      string
		fileCount int
	}{
		{name: "files=5", fileCount: 5},
		{name: "files=20", fileCount: 20},
		{name: "files=50", fileCount: 50},
	} {
		dir := b.TempDir()
		buildModuleParseBenchFixture(b, dir, tc.fileCount)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				parsed, err := Run(context.Background(), dir, []string{"service", "environment", "region", "module"})
				if err != nil {
					b.Fatalf("Run() error = %v", err)
				}
				benchModuleParseResult = parsed
			}
		})
	}
}

func buildModuleParseBenchFixture(tb testing.TB, dir string, fileCount int) {
	tb.Helper()

	for i := range fileCount {
		content := fmt.Sprintf(`
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
`, i, i, i, i)

		path := filepath.Join(dir, fmt.Sprintf("bench_%02d.tf", i))
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			tb.Fatalf("write %s: %v", path, err)
		}
	}
}
