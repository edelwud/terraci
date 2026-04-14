package lockfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDocument_FormatsHashesMultiline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".terraform.lock.hcl")

	doc := &LockDocument{
		Providers: []LockProviderEntry{{
			Source:      "registry.terraform.io/hashicorp/aws",
			Version:     "5.2.0",
			Constraints: "~> 5.2",
			Hashes:      LockHashSet{"zh:bbb", "h1:aaa"},
		}},
	}

	if err := NewWriter().WriteDocument(path, doc); err != nil {
		t.Fatalf("WriteDocument() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written lock file: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "hashes      = [\n") {
		t.Fatalf("hashes should be multiline:\n%s", text)
	}
	if !strings.Contains(text, "\n    \"h1:aaa\",\n") {
		t.Fatalf("lock file missing indented h1 hash:\n%s", text)
	}
	if !strings.Contains(text, "\n    \"zh:bbb\",\n") {
		t.Fatalf("lock file missing indented zh hash:\n%s", text)
	}
}
