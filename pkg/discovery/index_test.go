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

func TestModuleIndex_ByName(t *testing.T) {
	t.Parallel()

	idx := NewModuleIndex(testModules())

	ec2 := idx.ByName("ec2")
	if len(ec2) != 2 { // ec2 base + ec2/rabbitmq indexed by base name
		t.Errorf("ByName(ec2) = %d, want 2", len(ec2))
	}

	vpc := idx.ByName("vpc")
	if len(vpc) != 2 { // stage + prod
		t.Errorf("ByName(vpc) = %d, want 2", len(vpc))
	}
}

func TestModuleIndex_BaseModules(t *testing.T) {
	t.Parallel()

	idx := NewModuleIndex(testModules())

	if got := len(idx.BaseModules()); got != 3 {
		t.Errorf("BaseModules() = %d, want 3", got)
	}
}

func TestModuleIndex_Submodules(t *testing.T) {
	t.Parallel()

	idx := NewModuleIndex(testModules())

	if got := len(idx.Submodules()); got != 1 {
		t.Errorf("Submodules() = %d, want 1", got)
	}
}

func TestModuleIndex_Filter(t *testing.T) {
	t.Parallel()

	idx := NewModuleIndex(testModules())

	stage := idx.Filter(func(m *Module) bool { return m.Get("environment") == "stage" })
	if len(stage) != 3 {
		t.Errorf("Filter(stage) = %d, want 3", len(stage))
	}
}

func TestModuleIndex_FindInContext(t *testing.T) {
	t.Parallel()

	modules := testModules()
	idx := NewModuleIndex(modules)

	vpc := modules[0] // platform/stage/eu-central-1/vpc
	found := idx.FindInContext("ec2", vpc)
	if found == nil {
		t.Fatal("FindInContext(ec2) = nil")
	}
	if found.ID() != "platform/stage/eu-central-1/ec2" {
		t.Errorf("FindInContext(ec2) = %q, want ec2", found.ID())
	}

	if notFound := idx.FindInContext("nonexistent", vpc); notFound != nil {
		t.Errorf("FindInContext(nonexistent) = %v, want nil", notFound)
	}
}
