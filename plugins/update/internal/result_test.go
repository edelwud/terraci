package updateengine

import "testing"

func TestUpdateResultBuilder_Build(t *testing.T) {
	builder := NewUpdateResultBuilder()
	builder.AddModuleUpdate(ModuleVersionUpdate{Status: StatusUpdateAvailable})
	builder.AddProviderUpdate(ProviderVersionUpdate{Status: StatusApplied})
	builder.RecordError()

	result := builder.Build()
	if result == nil {
		t.Fatal("Build() returned nil")
	}
	if len(result.Modules) != 1 {
		t.Fatalf("modules = %d, want 1", len(result.Modules))
	}
	if len(result.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(result.Providers))
	}
	if result.Summary.TotalChecked != 2 {
		t.Errorf("TotalChecked = %d, want 2", result.Summary.TotalChecked)
	}
	if result.Summary.UpdatesAvailable != 1 {
		t.Errorf("UpdatesAvailable = %d, want 1", result.Summary.UpdatesAvailable)
	}
	if result.Summary.UpdatesApplied != 1 {
		t.Errorf("UpdatesApplied = %d, want 1", result.Summary.UpdatesApplied)
	}
	if result.Summary.Errors != 1 {
		t.Errorf("Errors = %d, want 1", result.Summary.Errors)
	}
}
