package aws

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
)

func TestResourceRegistrationsUnique(t *testing.T) {
	seen := make(map[ResourceKey]bool, len(Definition.Resources))
	for _, registration := range Definition.Resources {
		key := ResourceKey(registration.Type)
		if key == "" {
			t.Fatal("resource registration key must not be empty")
		}
		if registration.Handler == nil {
			t.Fatalf("resource registration %q has nil handler", key)
		}
		if seen[key] {
			t.Fatalf("duplicate resource registration: %q", key)
		}
		seen[key] = true
	}
}

func TestProviderRegisterHandlersUsesCatalog(t *testing.T) {
	registry := handler.NewRegistry()
	cloud.RegisterDefinitionHandlers(registry, Definition)

	for _, registration := range Definition.Resources {
		resolved, ok := registry.ResolveHandler(Definition.Manifest.ID, registration.Type)
		if !ok {
			t.Fatalf("resource %q was not registered", registration.Type)
		}
		if resolved == nil {
			t.Fatalf("resource %q has nil resolved handler", registration.Type)
		}
	}
}

func TestDefinitionContainsManifest(t *testing.T) {
	if Definition.Manifest.ID != awskit.ProviderID {
		t.Fatalf("Definition.Manifest.ID = %q, want %q", Definition.Manifest.ID, awskit.ProviderID)
	}
	if len(Definition.Resources) == 0 {
		t.Fatal("Definition.Resources must not be empty")
	}
}
