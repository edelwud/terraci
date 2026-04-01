package fsstore

import (
	"path/filepath"
	"testing"
)

func TestNamespaceAndBlobKeyValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "empty namespace rejected",
			fn: func() error {
				_, err := ParseNamespace("")
				return err
			},
		},
		{
			name: "absolute namespace rejected",
			fn: func() error {
				_, err := ParseNamespace("/tmp")
				return err
			},
		},
		{
			name: "escaping key rejected",
			fn: func() error {
				_, err := ParseBlobKey("../secret")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.fn(); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestNestedPathLayoutResolveAndDecode(t *testing.T) {
	t.Parallel()

	rootDir := filepath.Join(string(filepath.Separator), "tmp", "blobs")
	layout := NewNestedPathLayout(rootDir)
	namespace, err := ParseNamespace("cost/pricing")
	if err != nil {
		t.Fatalf("ParseNamespace() error = %v", err)
	}
	key, err := ParseBlobKey("aws/AmazonEC2/us-east-1.json")
	if err != nil {
		t.Fatalf("ParseBlobKey() error = %v", err)
	}

	paths, err := layout.Resolve(namespace, key)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	wantDataPath := filepath.Join(rootDir, "cost", "pricing", "aws", "AmazonEC2", "us-east-1.json")
	if paths.DataPath != wantDataPath {
		t.Fatalf("Resolve().DataPath = %q, want %q", paths.DataPath, wantDataPath)
	}

	decoded, err := layout.DecodeListedObject(namespace, wantDataPath)
	if err != nil {
		t.Fatalf("DecodeListedObject() error = %v", err)
	}
	if decoded.Value() != key.Value() {
		t.Fatalf("DecodeListedObject() = %q, want %q", decoded.Value(), key.Value())
	}
}
