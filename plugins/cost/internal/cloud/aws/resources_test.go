package aws

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
)

func TestResourceRegistrationsUnique(t *testing.T) {
	seen := make(map[awskit.ResourceKey]bool, len(definition.Resources))
	for _, registration := range definition.Resources {
		key := awskit.ResourceKey(registration.Type)
		if key == "" {
			t.Fatal("resource registration key must not be empty")
		}
		if err := registration.Definition.Validate(); err != nil {
			t.Fatalf("resource registration %q is invalid: %v", key, err)
		}
		if seen[key] {
			t.Fatalf("duplicate resource registration: %q", key)
		}
		seen[key] = true
	}
}

func TestDefinitionContainsManifest(t *testing.T) {
	if definition.Manifest.ID != awskit.ProviderID {
		t.Fatalf("Definition.Manifest.ID = %q, want %q", definition.Manifest.ID, awskit.ProviderID)
	}
	if len(definition.Resources) == 0 {
		t.Fatal("Definition.Resources must not be empty")
	}
}
