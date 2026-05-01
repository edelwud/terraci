package discovery

import "testing"

func TestModuleIndex_All(t *testing.T) {
	t.Parallel()

	modules := testModules()
	idx := NewModuleIndex(modules)

	if len(idx.All()) != 4 {
		t.Errorf("All() = %d, want 4", len(idx.All()))
	}
}

func TestModuleIndex_ByID(t *testing.T) {
	t.Parallel()

	idx := NewModuleIndex(testModules())

	if m := idx.ByID("platform/stage/eu-central-1/vpc"); m == nil {
		t.Error("ByID(vpc) = nil")
	}
	if m := idx.ByID("platform/stage/eu-central-1/ec2/rabbitmq"); m == nil {
		t.Error("ByID(ec2/rabbitmq) = nil")
	} else if !m.IsSubmodule() {
		t.Error("ec2/rabbitmq should be submodule")
	}
	if m := idx.ByID("nonexistent"); m != nil {
		t.Errorf("ByID(nonexistent) = %v, want nil", m)
	}
}

func TestModuleIndex_ByPath(t *testing.T) {
	t.Parallel()

	idx := NewModuleIndex(testModules())

	if m := idx.ByPath("/test/platform/stage/eu-central-1/vpc"); m == nil {
		t.Error("ByPath(vpc) = nil")
	}
}
