package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

// createModuleTree creates directories with main.tf files for testing.
func createModuleTree(t *testing.T, root string, paths []string) {
	t.Helper()
	for _, p := range paths {
		dir := filepath.Join(root, p)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# test"), 0o644); err != nil {
			t.Fatalf("write %s/main.tf: %v", dir, err)
		}
	}
}

// testModules returns a standard set of modules for index tests.
func testModules() []*Module {
	vpc := TestModule("platform", "stage", "eu-central-1", "vpc")
	vpc.Path = "/test/platform/stage/eu-central-1/vpc"

	ec2 := TestModule("platform", "stage", "eu-central-1", "ec2")
	ec2.Path = "/test/platform/stage/eu-central-1/ec2"

	rabbitmq := TestModule("platform", "stage", "eu-central-1", "ec2")
	rabbitmq.SetComponent("submodule", "rabbitmq")
	rabbitmq.Path = "/test/platform/stage/eu-central-1/ec2/rabbitmq"
	rabbitmq.RelativePath = "platform/stage/eu-central-1/ec2/rabbitmq"

	vpcProd := TestModule("platform", "prod", "eu-central-1", "vpc")
	vpcProd.Path = "/test/platform/prod/eu-central-1/vpc"

	return []*Module{vpc, ec2, rabbitmq, vpcProd}
}
