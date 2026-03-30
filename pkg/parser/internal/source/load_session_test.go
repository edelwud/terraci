package source

import (
	"context"
	"testing"
)

func TestLoadSessionRun(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"a.tf": `locals { service = "platform" }`,
		"b.tf": `module "vpc" { source = "../_modules/vpc" }`,
	})

	index, err := newLoadSession(dir).Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := len(index.Files()); got != 2 {
		t.Fatalf("files = %d, want 2", got)
	}
	if got := len(index.LocalsBlocks()); got != 1 {
		t.Fatalf("locals blocks = %d, want 1", got)
	}
	if got := len(index.ModuleBlocks()); got != 1 {
		t.Fatalf("module blocks = %d, want 1", got)
	}
}
